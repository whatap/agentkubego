package kubeapi

import (
	"context"
	"fmt"
	"github.com/whatap/kube/cadvisor/pkg/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net/http"
	"os"
	"time"
)

// injectExecTimeout bounds one container-ID exec (`<agent> -c <id>`, or a short JVM run for the Java agent).
const injectExecTimeout = 30 * time.Second

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

	// The caller holds the pod cache lock during this exec, so a hung stream must not block it forever.
	ctx, cancel := context.WithTimeout(context.Background(), injectExecTimeout)
	defer cancel()
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
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
