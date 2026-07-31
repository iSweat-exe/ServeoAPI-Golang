package docker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"serveoapi/internal/core/config"
	"serveoapi/internal/core/middleware"
	"serveoapi/internal/core/response"
	"serveoapi/internal/modules/v2/auth"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
)

type TerminalControlMessage struct {
	Type string `json:"type"` // "input" or "resize"
	Data string `json:"data,omitempty"`
	Cols uint   `json:"cols,omitempty"`
	Rows uint   `json:"rows,omitempty"`
}

// newUpgrader construit un upgrader qui n'accepte que les origines configurées.
// Une requête sans en-tête Origin (client non navigateur) reste acceptée : elle
// est de toute façon authentifiée par un ticket à usage unique.
func newUpgrader(cfg *config.Config) websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == "" || cfg.IsOriginAllowed(origin)
		},
	}
}

// hasContainerWritePermission vérifie les droits portés par le token du ticket WebSocket.
func hasContainerWritePermission(tokenString string) bool {
	_, permissions, err := middleware.ValidateToken(tokenString)
	if err != nil {
		return false
	}

	return middleware.HasPermission(permissions, "docker.containers.write")
}

// probeShell vérifie si /bin/bash est exécutable dans le conteneur
func probeShell(
	ctx context.Context,
	cli *client.Client,
	containerID string,
) string {
	execConfig := container.ExecOptions{ // Remarque : types.ExecConfig est obsolète, on utilise container.ExecOptions
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          []string{"test", "-x", "/bin/bash"},
	}

	resp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "/bin/sh"
	}

	err = cli.ContainerExecStart(ctx, resp.ID, container.ExecStartOptions{})
	if err != nil {
		return "/bin/sh"
	}

	// Boucler jusqu'à la fin (max 2 secondes)
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "/bin/sh"
		case <-timeout:
			return "/bin/sh"
		case <-ticker.C:
			inspect, err := cli.ContainerExecInspect(ctx, resp.ID)
			if err == nil && !inspect.Running {
				if inspect.ExitCode == 0 {
					return "/bin/bash"
				}
				return "/bin/sh"
			}
		}
	}
}

// TerminalHandler godoc
// @Summary      Interactive Docker Terminal (WebSocket)
// @Description  Upgrades connection to a WebSocket. Un ticket court (généré via /v2/auth/ticket) doit être fourni en paramètre d'URL (?ticket=...). Outputs raw binary frames.
// @Tags         docker
// @Param        id       path      string  true  "Container ID"
// @Param        ticket   query     string  true  "Authentication Ticket"
// @Success      101      {string}  string  "Switching Protocols to WebSocket"
// @Failure      400,401,403,404,500 {string} string
// @Router       /v2/docker/containers/{id}/exec [get]
func (h *Handler) TerminalHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	containerID := r.PathValue("id")
	if containerID == "" {
		response.SendError(w, http.StatusBadRequest, "Container ID is required")
		return
	}

	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		response.SendError(w, http.StatusUnauthorized, "Ticket is required")
		return
	}

	tokenString, ok := auth.ConsumeTicket(ticket)
	if !ok {
		response.SendError(w, http.StatusUnauthorized, "Invalid or expired ticket")
		return
	}

	if !hasContainerWritePermission(tokenString) {
		response.SendError(w, http.StatusForbidden, "Forbidden: Missing docker.containers.write")
		return
	}

	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "exec"
	}

	cli := h.Service.DockerCli

	upgrader := newUpgrader(h.Service.Config)
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Impossible d'écrire une erreur HTTP ici, upgrader s'en charge
	}
	defer ws.Close()

	// 3. Tester le Shell
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var hijackedResp types.HijackedResponse
	var errAttach error
	var resizeTarget string

	if mode == "console" {
		hijackedResp, errAttach = cli.ContainerAttach(ctx, containerID, container.AttachOptions{
			Stream: true,
			Stdin:  true,
			Stdout: true,
			Stderr: true,
		})
		if errAttach != nil {
			_ = ws.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(1011, "Container attach failed"),
				time.Now().Add(time.Second),
			)
			return
		}
		resizeTarget = containerID
	} else {
		shell := probeShell(ctx, cli, containerID)

		execConfig := container.ExecOptions{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          true,
			Cmd:          []string{shell},
		}

		execResp, errExec := cli.ContainerExecCreate(ctx, containerID, execConfig)
		if errExec != nil {
			_ = ws.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(1011, "Exec create failed"),
				time.Now().Add(time.Second),
			)
			return
		}

		hijackedResp, errAttach = cli.ContainerExecAttach(
			ctx,
			execResp.ID,
			container.ExecStartOptions{Tty: true},
		)
		if errAttach != nil {
			_ = ws.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(1011, "Exec attach failed"),
				time.Now().Add(time.Second),
			)
			return
		}
		resizeTarget = execResp.ID
	}

	// CRITIQUE : Nettoyage pour éviter les fuites de ressources
	// Si le handler se termine, tout est fermé.
	defer hijackedResp.Close()

	// Utiliser des channels pour attendre la première goroutine qui se termine
	errChan := make(chan error, 2)

	// Goroutine 1 : Lecture depuis WebSocket (Contrôles JSON) -> Écriture vers Docker Stdin & Redimensionnement
	go handleTerminalInput(
		ctx,
		ws,
		cli,
		resizeTarget,
		mode == "console",
		hijackedResp.Conn,
		errChan,
	)

	// Goroutine 2 : Lecture depuis Docker (Stream brut) -> Écriture vers WebSocket (Binaire)
	go handleTerminalOutput(ws, hijackedResp.Reader, errChan)

	// Attendre que la première goroutine se termine (soit l'utilisateur a fermé le WS, soit le shell s'est arrêté)
	err = <-errChan

	// Annule le contexte et ferme les flux pour débloquer immédiatement l'autre goroutine.
	cancel()
	_ = ws.SetReadDeadline(time.Now())
	hijackedResp.Close()

	// Si le shell s'est arrêté proprement (EOF), notifier le client WebSocket avant de fermer
	if err == nil {
		_ = ws.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Shell exited"),
			time.Now().Add(time.Second),
		)
	}

	// Les defer ws.Close() et hijackedResp.Close() (idempotent) nettoyent le reste.
}

func handleTerminalInput(
	ctx context.Context,
	ws *websocket.Conn,
	cli *client.Client,
	resizeTarget string,
	isConsole bool,
	conn io.Writer,
	errChan chan<- error,
) {
	for {
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
		}

		msgType, payload, err := ws.ReadMessage()
		if err != nil {
			errChan <- err
			return
		}

		if msgType == websocket.TextMessage {
			var ctrl TerminalControlMessage
			if err := json.Unmarshal(payload, &ctrl); err == nil {
				switch ctrl.Type {
				case "input":
					if _, err := conn.Write([]byte(ctrl.Data)); err != nil {
						errChan <- err
						return
					}
				case "resize":
					if isConsole {
						_ = cli.ContainerResize(ctx, resizeTarget, container.ResizeOptions{
							Height: ctrl.Rows,
							Width:  ctrl.Cols,
						})
					} else {
						_ = cli.ContainerExecResize(ctx, resizeTarget, container.ResizeOptions{
							Height: ctrl.Rows,
							Width:  ctrl.Cols,
						})
					}
				}
			}
		}
	}
}

func handleTerminalOutput(
	ws *websocket.Conn,
	reader io.Reader,
	errChan chan<- error,
) {
	buf := make([]byte, 8192)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if errWrite := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); errWrite != nil {
				errChan <- errWrite
				return
			}
		}
		if err != nil {
			if err == io.EOF {
				errChan <- nil // Sortie normale
			} else {
				errChan <- err
			}
			return
		}
	}
}
