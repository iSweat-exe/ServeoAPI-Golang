package docker

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"serveoapi/internal/core/config"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type DockerService struct {
	DockerCli *client.Client
	Config    *config.Config
}

func (s *DockerService) ListContainers(
	ctx context.Context,
) ([]ContainerInfo, error) {
	containers, err := s.DockerCli.ContainerList(
		ctx,
		container.ListOptions{All: true},
	)
	if err != nil {
		return nil, err
	}

	var resp []ContainerInfo
	for _, c := range containers {
		id := c.ID
		if len(id) > 12 {
			id = id[:12]
		}

		ports := make(map[string]string)
		for _, p := range c.Ports {
			if p.PublicPort != 0 {
				ports[strconv.Itoa(int(p.PublicPort))] = strconv.Itoa(int(p.PrivatePort))
			}
		}

		resp = append(resp, ContainerInfo{
			ID:     id,
			Names:  c.Names,
			Image:  c.Image,
			State:  c.State,
			Status: c.Status,
			Labels: c.Labels,
			Ports:  ports,
		})
	}
	if resp == nil {
		resp = []ContainerInfo{}
	}
	return resp, nil
}

func (s *DockerService) InspectContainer(
	ctx context.Context,
	id string,
) (container.InspectResponse, error) {
	return s.DockerCli.ContainerInspect(ctx, id)
}

func (s *DockerService) ActionContainer(
	ctx context.Context,
	id string,
	action string,
) error {
	switch action {
	case "start":
		return s.DockerCli.ContainerStart(ctx, id, container.StartOptions{})
	case "stop":
		return s.DockerCli.ContainerStop(ctx, id, container.StopOptions{})
	case "restart":
		return s.DockerCli.ContainerRestart(ctx, id, container.StopOptions{})
	default:
		return errors.New("invalid action")
	}
}

func (s *DockerService) DeleteContainer(
	ctx context.Context,
	id string,
	force bool,
) error {
	return s.DockerCli.ContainerRemove(
		ctx,
		id,
		container.RemoveOptions{
			Force:         force,
			RemoveVolumes: false,
		},
	)
}

func (s *DockerService) CreateContainer(
	ctx context.Context,
	req CreateContainerRequest,
) (ContainerInfo, error) {
	allowedRoot := s.Config.AllowedMountRoot
	var hostConfig container.HostConfig
	hostConfig.Binds = []string{}

	for _, v := range req.Volumes {
		parts := strings.SplitN(v, ":", 2)
		src := parts[0]

		if strings.Contains(src, "/") || strings.Contains(src, "\\") {
			if !strings.HasPrefix(src, allowedRoot) {
				return ContainerInfo{}, errors.New(
					"Security Error: Bind mounts are restricted to " + allowedRoot,
				)
			}
		}
		hostConfig.Binds = append(hostConfig.Binds, v)
	}

	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	for hostPort, containerPort := range req.Ports {
		port, err := nat.NewPort("tcp", containerPort)
		if err == nil {
			exposedPorts[port] = struct{}{}
			portBindings[port] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: hostPort,
				},
			}
		}
	}
	hostConfig.PortBindings = portBindings

	if req.RestartPolicy != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{
			Name: container.RestartPolicyMode(req.RestartPolicy),
		}
	}

	containerConfig := &container.Config{
		Image:        req.Image,
		Env:          req.Env,
		ExposedPorts: exposedPorts,
		Labels:       req.Labels,
	}

	resp, err := s.DockerCli.ContainerCreate(
		ctx,
		containerConfig,
		&hostConfig,
		nil,
		nil,
		req.Name,
	)
	if err != nil {
		if strings.Contains(err.Error(), "No such image") ||
			strings.Contains(err.Error(), "not found") {
			// Pull de l'image automatisé si manquante
			pullResp, pullErr := s.DockerCli.ImagePull(ctx, req.Image, image.PullOptions{})
			if pullErr != nil {
				return ContainerInfo{}, pullErr
			}
			// Bloquer jusqu'à ce que le pull soit terminé (lire tout le flux)
			_, _ = io.Copy(io.Discard, pullResp)
			pullResp.Close()

			// Retenter la création
			resp, err = s.DockerCli.ContainerCreate(
				ctx,
				containerConfig,
				&hostConfig,
				nil,
				nil,
				req.Name,
			)
			if err != nil {
				return ContainerInfo{}, err
			}
		} else {
			return ContainerInfo{}, err
		}
	}

	if err := s.DockerCli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return ContainerInfo{}, err
	}

	inspect, err := s.DockerCli.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return ContainerInfo{}, err
	}

	return ContainerInfo{
		ID:     inspect.ID[:12],
		Names:  []string{inspect.Name},
		Image:  inspect.Config.Image,
		State:  inspect.State.Status,
		Status: inspect.State.Status,
		Labels: inspect.Config.Labels,
	}, nil
}

func (s *DockerService) UpdateContainer(
	ctx context.Context,
	id string,
	req UpdateContainerRequest,
) (ContainerInfo, error) {
	oldContainer, err := s.DockerCli.ContainerInspect(ctx, id)
	if err != nil {
		return ContainerInfo{}, err
	}

	newConfig := oldContainer.Config
	newHostConfig := oldContainer.HostConfig
	newName := strings.TrimPrefix(oldContainer.Name, "/")

	if len(req.Env) > 0 {
		newConfig.Env = req.Env
	}
	if req.Memory > 0 {
		newHostConfig.Resources.Memory = req.Memory
	}
	if req.Ports != nil {
		portBindings := nat.PortMap{}
		exposedPorts := nat.PortSet{}
		for hostPort, containerPort := range req.Ports {
			port, err := nat.NewPort("tcp", containerPort)
			if err == nil {
				exposedPorts[port] = struct{}{}
				portBindings[port] = []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: hostPort,
					},
				}
			}
		}
		newConfig.ExposedPorts = exposedPorts
		newHostConfig.PortBindings = portBindings
	}

	_ = s.DockerCli.ContainerStop(ctx, id, container.StopOptions{})
	err = s.DockerCli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
	if err != nil {
		return ContainerInfo{}, err
	}

	importNetworkConfig := &network.NetworkingConfig{
		EndpointsConfig: oldContainer.NetworkSettings.Networks,
	}

	resp, err := s.DockerCli.ContainerCreate(
		ctx,
		newConfig,
		newHostConfig,
		importNetworkConfig,
		nil,
		newName,
	)
	if err != nil {
		return ContainerInfo{}, err
	}

	if err := s.DockerCli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return ContainerInfo{}, err
	}

	return ContainerInfo{
		ID:     resp.ID[:12],
		Names:  []string{"/" + newName},
		Image:  newConfig.Image,
		State:  "running",
		Status: "running",
		Labels: newConfig.Labels,
	}, nil
}
