package client

import (
	"github.com/docker/docker/client"
	"github.com/whatap/kube/tools/util/logutil"
	"sync"
)

var (
	dockerClient *client.Client
	mu           = sync.Mutex{}
)

func GetDockerClient() (*client.Client, error) {
	mu.Lock()
	defer mu.Unlock()
	if dockerClient == nil {
		dockerClientThisTime, err := client.NewClientWithOpts(client.WithVersion("1.40"))

		if err != nil {
			logutil.Errorf("execDebug", "GetDockerClientErr=%v", err)
			return nil, err
		}
		dockerClient = dockerClientThisTime
	}
	return dockerClient, nil
}
