package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"serveoapi/internal/modules/v2/auth"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for the panel
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

// checkPermission from JWT claims
func hasContainerWritePermission(tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return auth.JwtSecretKey, nil
	})

	if err != nil || !token.Valid {
		return false
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		permissions, _ := claims["permissions"].(string)
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
	}
	return false
}

// probeShell checks if /bin/bash is executable in the container
func probeShell(ctx context.Context, cli *client.Client, containerID string) string {
	execConfig := container.ExecOptions{ // Note: types.ExecConfig is deprecated, use container.ExecOptions in new SDK
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

	// Poll until finish (max 2 seconds)
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
func TerminalHandler(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Cannot write http error here anymore, upgrader handles it
	}
	defer ws.Close()

	// 1. First-Message Auth (5s timeout)
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

	// Remove the read deadline for normal operation
	ws.SetReadDeadline(time.Time{})

	// 2. Initialize Docker Client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "Docker client error"), time.Now().Add(time.Second))
		return
	}
	defer cli.Close()

	// 3. Probe Shell
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	shell := probeShell(ctx, cli, containerID)

	// 4. Create Interactive Exec
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true, // Docker merges this into stdout when Tty=true
		Tty:          true,
		Cmd:          []string{shell},
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "Exec create failed"), time.Now().Add(time.Second))
		return
	}

	// 5. Attach to Exec
	hijackedResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{Tty: true})
	if err != nil {
		ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "Exec attach failed"), time.Now().Add(time.Second))
		return
	}
	
	// CRITICAL: Cleanup to prevent resource leaks
	// If the handler exits, everything is closed.
	defer hijackedResp.Close()

	// Use channels to wait for the first goroutine that finishes
	errChan := make(chan error, 2)

	// Goroutine 1: Read from WebSocket (JSON Controls) -> Write to Docker Stdin & Resize
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

	// Goroutine 2: Read from Docker Raw Stream -> Write to WebSocket (Binary)
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
					errChan <- nil // Normal exit
				} else {
					errChan <- err
				}
				return
			}
		}
	}()

	// Wait for the first goroutine to exit (either user closed WS, or shell exited)
	err = <-errChan

	// If the shell exited gracefully (EOF), notify the WebSocket client before closing
	if err == nil {
		ws.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Shell exited"),
			time.Now().Add(time.Second),
		)
	}

	// The deferred ws.Close() and hijackedResp.Close() will now execute,
	// forcefully unblocking the other goroutine and cleaning up all resources.
}
