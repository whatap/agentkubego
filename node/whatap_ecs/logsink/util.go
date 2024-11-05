package logsink

import (
	"context"

	"github.com/docker/docker/api/types"
	whatap_docker "github.com/whatap/kube/node/src/whatap/util/docker"
)

func findAllContainersOnNode(onContainerDetected func(types.ContainerJSON)) error {
	cli, err := whatap_docker.GetDockerClient()
	if err != nil {
		return err
	}
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, c := range containers {

		inspectContainer, err := cli.ContainerInspect(context.Background(), c.ID)

		if err != nil {
			return err
		}
		onContainerDetected(inspectContainer)

	}
	return nil
}
