package run

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"io/ioutil"

	"github.com/docker/go-connections/nat"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)
//This Go code defines a function createServer that uses the Docker SDK for Go to create and start a Docker container. 
func createServer(verbose bool, image string, port string, args []string, env []string, name string, volumes []string) (string, error) {  //
	log.Printf("Creating server using %s...\n", image)
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
	}else {
		_, err := io.Copy(ioutil.Discard, reader)
		if err != nil {
			log.Printf("WARNING: couldn't get docker output\n%+v", err)
		}
	}
	containerLabels := make(map[string]string)
	containerLabels["app"] = "k3d"
	containerLabels["component"] = "server"
	containerLabels["created"] = time.Now().Format("2006-01-02 15:04:05")
	containerLabels["cluster"] = name

	containerName := fmt.Sprintf("k3d-%s-server", name)

	containerPort := nat.Port(fmt.Sprintf("%s/tcp", port))

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: port,
				},
			},
		},
		Privileged: true,
	}
	if len(volumes) > 0 && volumes[0] != "" {
		hostConfig.Binds = volumes
	}
	//?kichu buji nai
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			name: &network.EndpointSettings{
				Aliases: []string{containerName},
			},
		},
	}

	resp, err := docker.ContainerCreate(ctx, &container.Config{
		Image: image,
		Cmd:   append([]string{"server"}, args...),
		ExposedPorts: nat.PortSet{
			containerPort: struct{}{},
		},
		Env:    env,
		Labels: containerLabels,
	}, hostConfig, networkingConfig,nil, containerName)
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create container %s\n%+v", containerName, err)
	}

	if err := docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("ERROR: couldn't start container %s\n%+v", containerName, err)
	}

	return resp.ID, nil
}

func createWorker(verbose bool, image string, args []string, env []string, name string, volumes []string, postfix string, serverPort string) (string, error) {
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
		_, err := io.Copy(os.Stdout, reader)
		if err != nil {
			log.Printf("WARNING: couldn't get docker output\n%+v", err)
		}
	}

	containerLabels := make(map[string]string)
	containerLabels["app"] = "k3d"
	containerLabels["component"] = "worker"
	containerLabels["created"] = time.Now().Format("2006-01-02 15:04:05")
	containerLabels["cluster"] = name

	containerName := fmt.Sprintf("k3d-%s-worker-%s", name, postfix)

	env = append(env, fmt.Sprintf("K3S_URL=https://k3d-%s-server:%s", name, serverPort))

	hostConfig := &container.HostConfig{
		Tmpfs: map[string]string{
			"/run":     "",
			"/var/run": "",
		},
		Privileged: true,
	}

	if len(volumes) > 0 && volumes[0] != "" {
		hostConfig.Binds = volumes
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			name: &network.EndpointSettings{
				Aliases: []string{containerName},
			},
		},
	}
	resp, err := docker.ContainerCreate(ctx, &container.Config{
		Image:  image,
		Env:    env,
		Labels: containerLabels,
	}, hostConfig, networkingConfig,nil, containerName)
	if err != nil {
		return "", fmt.Errorf("ERROR: couldn't create container %s\n%+v", containerName, err)
	}

	if err := docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("ERROR: couldn't start container %s\n%+v", containerName, err)
	}

	return resp.ID, nil
}

func removeContainer(ID string) error {
	ctx := context.Background()
	docker, err := client.NewEnvClient()
	if err != nil {
		return fmt.Errorf("ERROR: couldn't create docker client\n%+v", err)
	}
	if err := docker.ContainerRemove(ctx, ID, container.RemoveOptions{}); err != nil {
		log.Printf("WARNING: couldn't delete container [%s], trying a force remove now.", ID)
		if err := docker.ContainerRemove(ctx, ID, container.RemoveOptions{Force: true}); err != nil {
			return fmt.Errorf("FAILURE: couldn't delete container [%s] -> %+v", ID, err)
		}
	}
	return nil
}
