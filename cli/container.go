package run

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/go-connections/nat"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)
//This Go code defines a function createServer that uses the Docker SDK for Go to create and start a Docker container. 
func createServer(verbose bool, image string, port string, args []string, env []string, name string, ) (string, error) {  //volumes []string
	ctx := context.Background()
	docker, err := client.NewEnvClient()
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create docker client\n%+v", err)
	}

	reader, err := docker.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't pull image %s\n%+v", image, err)
	}
	if verbose {
		_, err := io.Copy(os.Stdout, reader) // TODO: only if verbose mode
		if err != nil {
			log.Printf("WARNING: couldn't get docker output\n%+v", err)
		}
	}//docker er output niye asteche

	containerLabels := make(map[string]string)
	containerLabels["app"] = "k3d"
	containerLabels["component"] = "server"
	containerLabels["created"] = time.Now().Format("2006-01-02 15:04:05")
	containerLabels["cluster"] = name

	containerName := fmt.Sprintf("%s-server", name)

	containerPort := nat.Port(fmt.Sprintf("%s/tcp", port))

	resp, err := docker.ContainerCreate(ctx, &container.Config{
		Image: image,
		Cmd:   append([]string{"server"}, args...),
		ExposedPorts: nat.PortSet{
			containerPort: struct{}{},
		},
		Env:    env,
		Labels: containerLabels,
	}, &container.HostConfig{
		//Binds: volumes,
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: port,
				},
			},
		},
		Privileged: true,
	}, nil,nil, containerName)
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create container %s\n%+v", containerName, err)
	}

	if err := docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("ERROR: couldn't start container %s\n%+v", containerName, err)
	}

	return resp.ID, nil

}
