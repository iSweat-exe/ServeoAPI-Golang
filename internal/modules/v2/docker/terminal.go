package docker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"serveoapi/internal/core/middleware"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Autoriser toutes les origines pour le panel
	},
}

type TerminalAuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
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
func probeShell(ctx context.Context, cli *client.Client, containerID string) string {
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
// @Description  Upgrades connection to a WebSocket. Expects first message: {"type": "auth", "token": "JWT"}. Subsequent JSON messages for input/resize. Outputs raw binary frames.
// @Tags         docker
// @Param        id   path      string  true  "Container ID"
// @Success      101  {string}  string  "Switching Protocols to WebSocket"
// @Failure      400,401,403,404,500 {string} string
// @Router       /v2/docker/containers/{id}/exec [get]
func (h *Handler) TerminalHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	if containerID == "" {
		http.Error(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	cli := h.Service.DockerCli

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Impossible d'écrire une erreur HTTP ici, upgrader s'en charge
	}
	defer ws.Close()

	// 1. Authentification au premier message (timeout 5s)
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msgBytes, err := ws.ReadMessage()
	if err != nil {
		ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "Auth timeout or error"), time.Now().Add(time.Second))
		return
	}

	var authMsg TerminalAuthMessage
	if err := json.Unmarshal(msgBytes, &authMsg); err != nil || authMsg.Type != "auth" {
		ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "Expected auth message"), time.Now().Add(time.Second))
		return
	}

	if !hasContainerWritePermission(authMsg.Token) {
		ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4003, "Forbidden: Missing docker.containers.write"), time.Now().Add(time.Second))
		return
	}

	// Supprimer la deadline de lecture pour le fonctionnement normal
	ws.SetReadDeadline(time.Time{})

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
		ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "Exec create failed"), time.Now().Add(time.Second))
		return
	}

	// 5. S'attacher à l'Exec
	hijackedResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{Tty: true})
	if err != nil {
		ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "Exec attach failed"), time.Now().Add(time.Second))
		return
	}

	// CRITIQUE : Nettoyage pour éviter les fuites de ressources
	// Si le handler se termine, tout est fermé.
	defer hijackedResp.Close()

	// Utiliser des channels pour attendre la première goroutine qui se termine
	errChan := make(chan error, 2)

	// Goroutine 1 : Lecture depuis WebSocket (Contrôles JSON) -> Écriture vers Docker Stdin & Redimensionnement
	go func() {
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
						_, err := hijackedResp.Conn.Write([]byte(ctrl.Data))
						if err != nil {
							errChan <- err
							return
						}
					case "resize":
						_ = cli.ContainerExecResize(ctx, execResp.ID, container.ResizeOptions{
							Height: ctrl.Rows,
							Width:  ctrl.Cols,
						})
					}
				}
			}
		}
	}()

	// Goroutine 2 : Lecture depuis Docker (Stream brut) -> Écriture vers WebSocket (Binaire)
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := hijackedResp.Reader.Read(buf)
			if n > 0 {
				errWrite := ws.WriteMessage(websocket.BinaryMessage, buf[:n])
				if errWrite != nil {
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
	}()

	// Attendre que la première goroutine se termine (soit l'utilisateur a fermé le WS, soit le shell s'est arrêté)
	err = <-errChan

	// Si le shell s'est arrêté proprement (EOF), notifier le client WebSocket avant de fermer
	if err == nil {
		ws.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Shell exited"),
			time.Now().Add(time.Second),
		)
	}

	// Les defer ws.Close() et hijackedResp.Close() vont maintenant s'exécuter,
	// débloquant de force l'autre goroutine et nettoyant toutes les ressources.
}
