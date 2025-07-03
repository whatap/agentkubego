package client

import (
	"context"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/whatap/kube/tools/util/logutil"
)

var (
	containerdClient     *containerd.Client
	containerdNamespaces []string
)

func GetContainerdClient() (*containerd.Client, error) {
	mu.Lock()
	defer mu.Unlock()
	if containerdClient == nil {
		logutil.Infof("GetContainerdClient", "Initializing containerd client connection to /run/containerd/containerd.sock")
		newContainerdClient, err := containerd.New("/run/containerd/containerd.sock")
		if err != nil {
			logutil.Errorf("GetContainerdClient", "Failed to connect to containerd socket: %v", err)
			return nil, err
		}

		containerdClient = newContainerdClient
		logutil.Infof("GetContainerdClient", "Successfully connected to containerd")

		if nss, err := containerdClient.NamespaceService().List(context.Background()); err == nil {
			containerdNamespaces = nss
			logutil.Infof("GetContainerdClient", "Found %d containerd namespaces: %v", len(containerdNamespaces), containerdNamespaces)
		} else {
			logutil.Errorf("GetContainerdClient", "Failed to list containerd namespaces: %v", err)
		}
	} else {
		logutil.Infof("GetContainerdClient", "Reusing existing containerd client")
	}
	return containerdClient, nil
}

func LoadContainerD(containerid string) (containerd.Container, context.Context, error) {
	logutil.Infof("LoadContainerD", "Loading container: %s", containerid)
	cli, err := GetContainerdClient()
	if err != nil {
		logutil.Errorf("LoadContainerD", "Failed to get containerd client: %v", err)
		return nil, nil, err
	}

	logutil.Infof("LoadContainerD", "Searching for container %s across %d namespaces", containerid, len(containerdNamespaces))
	for _, containerdNamespace := range containerdNamespaces {
		logutil.Infof("LoadContainerD", "Checking namespace: %s for container: %s", containerdNamespace, containerid)
		ctx := namespaces.WithNamespace(context.Background(), containerdNamespace)

		resp, err := cli.LoadContainer(ctx, containerid)
		if err == nil {
			logutil.Infof("LoadContainerD", "Successfully found container %s in namespace: %s", containerid, containerdNamespace)
			return resp, ctx, err
		} else {
			logutil.Infof("LoadContainerD", "Container %s not found in namespace %s: %v", containerid, containerdNamespace, err)
		}
	}
	logutil.Errorf("LoadContainerD", "Container %s not found in any namespace", containerid)
	return nil, nil, fmt.Errorf("container ", containerid, " not found")
}
