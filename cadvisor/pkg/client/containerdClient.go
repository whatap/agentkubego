package client

import (
	"context"
	"fmt"
	"strings"

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

func resetContainerdClient() {
	mu.Lock()
	defer mu.Unlock()
	if containerdClient != nil {
		containerdClient.Close()
		containerdClient = nil
		containerdNamespaces = nil
	}
}

func LoadContainerD(containerid string) (containerd.Container, context.Context, error) {
	cli, err := GetContainerdClient()
	if err != nil {
		return nil, nil, err
	}

	if len(containerdNamespaces) == 0 {
		logutil.Errorf("containerdClient", "no containerd namespaces available, resetting client")
		resetContainerdClient()
		return nil, nil, fmt.Errorf("no containerd namespaces available, retrying on next request")
	}

	var lastErr error
	for _, containerdNamespace := range containerdNamespaces {
		ctx := namespaces.WithNamespace(context.Background(), containerdNamespace)

		resp, err := cli.LoadContainer(ctx, containerid)
		if err == nil {
			return resp, ctx, err
		}
		lastErr = err
	}

	if lastErr != nil && isGrpcConnectionError(lastErr) {
		logutil.Errorf("containerdClient", "gRPC connection error detected, resetting client: %v", lastErr)
		resetContainerdClient()
		return nil, nil, fmt.Errorf("containerd connection lost, retrying on next request: %v", lastErr)
	}

	return nil, nil, fmt.Errorf("container %s not found", containerid)
}

func isGrpcConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "transport is closing") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "EOF") ||
		strings.Contains(errMsg, "unavailable") ||
		strings.Contains(errMsg, "connection error")
}
