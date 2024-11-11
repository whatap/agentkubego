package kubeapi

import (
	"fmt"
	"github.com/whatap/kube/cadvisor/pkg/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net/http"
	"os"
)

func ExecCommandForInjectContainerIdToWhatapAgent(namespace string, podName string, containerName string, containerId string, cmds []string) error {
	kubeClient, err := client.GetKubernetesClient()
	if err != nil {
		fmt.Printf("InitializeInformer error getting client: %v\n", err)
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	req := kubeClient.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   cmds,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, http.MethodPost, req.URL())
	if err != nil {
		return err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		return err
	}

	return nil
}
