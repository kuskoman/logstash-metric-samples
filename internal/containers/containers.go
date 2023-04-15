package containers

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	apiClient "github.com/kuskoman/logstash-metric-samples/internal/client"
	"github.com/kuskoman/logstash-metric-samples/internal/manager"
)

var containerManager = manager.NewContainerManager()

const logstashRegistry = "docker.elastic.co/logstash/logstash"

func RemoveContainer(ctx context.Context, dockerClient *client.Client, containerID string) {
	log.Printf("Removing container %s", containerID)
	err := dockerClient.ContainerStop(ctx, containerID, container.StopOptions{})
	handleRemoveContainerError(err, containerID)
	err = dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{})
	handleRemoveContainerError(err, containerID)
}

func handleRemoveContainerError(err error, containerID string) {
	if err == nil {
		return
	}

	log.Printf("Error removing container: %s", err)
	fmt.Printf("To remove container manually, run: docker rm %s", containerID)
}

func pullLogstashImage(ctx context.Context, dockerClient *client.Client, version string) (string, error) {
	imageName := fmt.Sprintf("%s:%s", logstashRegistry, version)
	log.Printf("Pulling image %s", imageName)
	pullResp, err := dockerClient.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}

	defer pullResp.Close()

	if _, err := io.Copy(io.Discard, pullResp); err != nil {
		return "", err
	}

	return imageName, nil
}

func createLogstashContainerConfig(imageName string, port int) (*container.Config, *container.HostConfig) {
	containerConfig := &container.Config{
		Image: imageName,
		ExposedPorts: nat.PortSet{
			"9600/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"9600/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: strconv.Itoa(port)},
			},
		},
	}

	return containerConfig, hostConfig
}

func SpawnLogstashContainer(ctx context.Context, dockerClient *client.Client, version string) (string, int, error) {
	imageName, err := pullLogstashImage(ctx, dockerClient, version)
	if err != nil {
		return "", 0, err
	}

	port, err := containerManager.AssignFreePort()
	if err != nil {
		return "", 0, err
	}

	containerName := fmt.Sprintf("logstash-%s", version)
	log.Printf("Creating container %s on port %d", containerName, port)

	containerConfig, hostConfig := createLogstashContainerConfig(imageName, port)

	resp, err := dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", 0, err
	}

	containerID := resp.ID

	if err := dockerClient.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return "", 0, err
	}

	return containerID, port, nil
}

func WaitForContainerToBeReady(ctx context.Context, port int) error {
	url := fmt.Sprintf("http://localhost:%d/_node/stats", port)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			data, _ := apiClient.GetJSONFromAPI(ctx, url)
			if data != nil {
				return nil
			}
			time.Sleep(1 * time.Second)
		}
	}
}
