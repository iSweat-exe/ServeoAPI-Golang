package docker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"serveoapi/internal/core/middleware"
	"serveoapi/internal/modules/v2/auth"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Autoriser toutes les origines pour le panel
	},
}

type TerminalControlMessage struct {
	Type string `json:"type"` // "input" or "resize"
	Data string `json:"data,omitempty"`
	Cols uint   `json:"cols,omitempty"`
	Rows uint   `json:"rows,omitempty"`
}

// checkPermission vérifie les droits depuis le token JWT
func hasContainerWritePermission(tokenString string) bool {
	_, permissions, err := middleware.ValidateToken(tokenString)
	if err != nil {
		return false
	}

	perms := strings.Split(permissions, ",")
	for _, p := range perms {
		p = strings.TrimSpace(p)
		if p == "*" || p == "docker.containers.write" {
			return true
		}
		if strings.HasSuffix(p, ".*") {
			prefix := strings.TrimSuffix(p, ".*")
			if strings.HasPrefix("docker.containers.write", prefix+".") {
				return true
			}
		}
	}
	return false
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
		http.Error(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		http.Error(w, "Ticket is required", http.StatusUnauthorized)
		return
	}

	tokenString, ok := auth.ConsumeTicket(ticket)
	if !ok {
		http.Error(w, "Invalid or expired ticket", http.StatusUnauthorized)
		return
	}

	if !hasContainerWritePermission(tokenString) {
		http.Error(w, "Forbidden: Missing docker.containers.write", http.StatusForbidden)
		return
	}

	cli := h.Service.DockerCli

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Impossible d'écrire une erreur HTTP ici, upgrader s'en charge
	}
	defer ws.Close()

	// 3. Tester le Shell
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	shell := probeShell(ctx, cli, containerID)

	// 4. Créer un Exec interactif
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true, // Docker fusionne ceci dans stdout quand Tty=true
		Tty:          true,
		Cmd:          []string{shell},
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		_ = ws.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(1011, "Exec create failed"),
			time.Now().Add(time.Second),
		)
		return
	}

	// 5. S'attacher à l'Exec
	hijackedResp, err := cli.ContainerExecAttach(
		ctx,
		execResp.ID,
		container.ExecStartOptions{Tty: true},
	)
	if err != nil {
		_ = ws.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(1011, "Exec attach failed"),
			time.Now().Add(time.Second),
		)
		return
	}

	// CRITIQUE : Nettoyage pour éviter les fuites de ressources
	// Si le handler se termine, tout est fermé.
	defer hijackedResp.Close()

	// Utiliser des channels pour attendre la première goroutine qui se termine
	errChan := make(chan error, 2)

	// Goroutine 1 : Lecture depuis WebSocket (Contrôles JSON) -> Écriture vers Docker Stdin & Redimensionnement
	go handleTerminalInput(ctx, ws, cli, execResp.ID, hijackedResp.Conn, errChan)

	// Goroutine 2 : Lecture depuis Docker (Stream brut) -> Écriture vers WebSocket (Binaire)
	go handleTerminalOutput(ws, hijackedResp.Reader, errChan)

	// Attendre que la première goroutine se termine (soit l'utilisateur a fermé le WS, soit le shell s'est arrêté)
	err = <-errChan

	// Si le shell s'est arrêté proprement (EOF), notifier le client WebSocket avant de fermer
	if err == nil {
		_ = ws.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Shell exited"),
			time.Now().Add(time.Second),
		)
	}

	// Les defer ws.Close() et hijackedResp.Close() vont maintenant s'exécuter,
	// débloquant de force l'autre goroutine et nettoyant toutes les ressources.
}

func handleTerminalInput(
	ctx context.Context,
	ws *websocket.Conn,
	cli *client.Client,
	execID string,
	conn io.Writer,
	errChan chan<- error,
) {
	for {
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
					_ = cli.ContainerExecResize(ctx, execID, container.ResizeOptions{
						Height: ctrl.Rows,
						Width:  ctrl.Cols,
					})
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
