package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ovh/go-ovh/ovh"

	"serveoapi/internal/core/config"
)

var (
	lastRebootMap = make(map[string]time.Time)
	cooldownMutex sync.Mutex
	cooldownFile  = "data/mcp_cooldowns.json"
	dockerClient  *client.Client
	ovhClient     *ovh.Client
)

func loadCooldowns() {
	cooldownMutex.Lock()
	defer cooldownMutex.Unlock()
	b, err := os.ReadFile(cooldownFile)
	if err == nil {
		json.Unmarshal(b, &lastRebootMap)
	}
}

func saveCooldowns() {
	cooldownMutex.Lock()
	defer cooldownMutex.Unlock()
	os.MkdirAll("data", 0755)
	b, err := json.Marshal(lastRebootMap)
	if err == nil {
		os.WriteFile(cooldownFile, b, 0644)
	}
}

func initClients(cfg *config.Config) {
	if cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err == nil {
		dockerClient = cli
	}
	if cfg.OvhEndpoint != "" {
		if oc, err := ovh.NewClient(cfg.OvhEndpoint, cfg.OvhAppKey, cfg.OvhAppSecret, cfg.OvhConsumerKey); err == nil {
			ovhClient = oc
		}
	}
}

func registerTools(cfg *config.Config) {
	initClients(cfg)
	loadCooldowns()

	registerDockerListTool()
	registerDockerLogsTool()
	registerFileReadTool(cfg)
	registerOvhRebootTool()
}

func registerDockerListTool() {
	mcpServer.AddTool(mcp.NewTool("docker_list",
		mcp.WithDescription("List all docker containers with their statuses"),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := hasPermission(ctx, "docker.read"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if dockerClient == nil {
			return mcp.NewToolResultError("Docker client unavailable"), nil
		}

		containers, err := dockerClient.ContainerList(ctx, container.ListOptions{All: true})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list containers: %v", err)), nil
		}

		result := "Containers:\n"
		for _, c := range containers {
			name := "unknown"
			if len(c.Names) > 0 {
				name = c.Names[0]
			}
			result += fmt.Sprintf("- Name: %s | Status: %s | State: %s\n", name, c.Status, c.State)
		}

		return mcp.NewToolResultText(result), nil
	})
}

func registerDockerLogsTool() {
	mcpServer.AddTool(mcp.NewTool("docker_logs",
		mcp.WithDescription("Read the recent logs of a docker container"),
		mcp.WithString("container_name", mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := hasPermission(ctx, "docker.read"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if dockerClient == nil {
			return mcp.NewToolResultError("Docker client unavailable"), nil
		}

		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Invalid arguments format"), nil
		}
		containerName, _ := args["container_name"].(string)

		options := container.LogsOptions{ShowStdout: true, ShowStderr: true, Tail: "100"}
		out, err := dockerClient.ContainerLogs(ctx, containerName, options)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get logs: %v", err)), nil
		}
		defer out.Close()

		logsBytes, err := io.ReadAll(out)
		if err != nil {
			return mcp.NewToolResultError("failed to read logs stream"), nil
		}

		return mcp.NewToolResultText(string(logsBytes)), nil
	})
}

func registerFileReadTool(cfg *config.Config) {
	mcpServer.AddTool(mcp.NewTool("file_read",
		mcp.WithDescription("Read a file inside a container's persistent data directory"),
		mcp.WithString("container_name", mcp.Required()),
		mcp.WithString("file_path", mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := hasPermission(ctx, "files.read"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Invalid arguments format"), nil
		}
		containerName, _ := args["container_name"].(string)
		reqPath, _ := args["file_path"].(string)

		rootPath := filepath.Join(cfg.AllowedMountRoot, containerName)

		root, err := os.OpenRoot(rootPath)
		if err != nil {
			return mcp.NewToolResultError("Cannot open root directory safely"), nil
		}
		defer root.Close()

		f, err := root.OpenFile(reqPath, os.O_RDONLY, 0)
		if err != nil {
			return mcp.NewToolResultError("File not found or cannot be read"), nil
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			return mcp.NewToolResultError("Failed to read file content"), nil
		}

		return mcp.NewToolResultText(string(content)), nil
	})
}

func registerOvhRebootTool() {
	mcpServer.AddTool(mcp.NewTool("ovh_reboot",
		mcp.WithDescription("Hard reboot an OVHcloud dedicated server"),
		mcp.WithString("service_name", mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := hasPermission(ctx, "ovh.write"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if ovhClient == nil {
			return mcp.NewToolResultError("OVH client unavailable"), nil
		}

		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Invalid arguments format"), nil
		}
		serviceName, _ := args["service_name"].(string)

		cooldownMutex.Lock()
		lastReboot, exists := lastRebootMap[serviceName]
		cooldownMutex.Unlock()

		if exists && time.Since(lastReboot) < 15*time.Minute {
			return mcp.NewToolResultError("cooldown active: cannot reboot this server again so soon"), nil
		}

		err := ovhClient.Post(fmt.Sprintf("/dedicated/server/%s/reboot", serviceName), nil, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to reboot server: %v", err)), nil
		}

		cooldownMutex.Lock()
		lastRebootMap[serviceName] = time.Now()
		cooldownMutex.Unlock()
		saveCooldowns()

		return mcp.NewToolResultText("Server reboot initiated successfully"), nil
	})
}
