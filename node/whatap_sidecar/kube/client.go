package kube

import (
	"k8s.io/client-go/kubernetes"
	k8srest "k8s.io/client-go/rest"
)

var (
	kubeClient *kubernetes.Clientset
)

func GetKubeClient() (*kubernetes.Clientset, error) {
	if kubeClient == nil {
		config, err := k8srest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		kubeClient = clientset
	}
	return kubeClient, nil
}
