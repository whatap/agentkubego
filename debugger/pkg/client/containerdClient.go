package client

import (
	"context"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
)

var (
	containerdClient     *containerd.Client
	containerdNamespaces []string
)

func GetContainerdClient() (*containerd.Client, error) {
	mu.Lock()
	defer mu.Unlock()
	if containerdClient == nil {
		newContainerdClient, err := containerd.New("/run/containerd/containerd.sock")
		if err != nil {
			return nil, err
		}

		containerdClient = newContainerdClient

		if nss, err := containerdClient.NamespaceService().List(context.Background()); err == nil {
			containerdNamespaces = nss

		}
	}
	return containerdClient, nil
}

func LoadContainerD(containerid string) (containerd.Container, context.Context, error) {
	cli, err := GetContainerdClient()
	if err != nil {
		return nil, nil, err
	}

	for _, containerdNamespace := range containerdNamespaces {
		ctx := namespaces.WithNamespace(context.Background(), containerdNamespace)

		resp, err := cli.LoadContainer(ctx, containerid)
		if err == nil {
			return resp, ctx, err
		}
	}
	return nil, nil, fmt.Errorf("container ", containerid, " not found")
}
