package docker

import (
	"log"
	"sync"

	"github.com/docker/docker/client"
)

var (
	dockerCli  *client.Client
	clientOnce sync.Once
)

// GetClient returns a singleton instance of the Docker client
func GetClient() *client.Client {
	clientOnce.Do(func() {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			log.Fatalf("Failed to initialize Docker client: %v", err)
		}
		dockerCli = cli
	})
	return dockerCli
}
