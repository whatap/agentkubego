package client

import (
	"github.com/whatap/kube/master/pkg/config"
	"k8s.io/client-go/kubernetes"
	k8srest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

var apiClient *kubernetes.Clientset

func GetKubernetesClient() (*kubernetes.Clientset, error, bool) {
	if apiClient != nil {
		return apiClient, nil, true
	}

	//load config from default config
	kubernetesClientConfig, err := k8srest.InClusterConfig()
	if err != nil {
		kubeConfig := config.Conf.KubeConfigPath
		kubernetesClientConfig, err = clientcmd.BuildConfigFromFlags(config.Conf.KubeMasterUrl, kubeConfig)
		if err != nil {
			log.Printf("error getting out-of-cluster config: %v\n", err)
			log.Printf("try load client from default cluster config . . .\n")
		}
	}

	//load client from default cluster config
	client, err := kubernetes.NewForConfig(kubernetesClientConfig)
	if err != nil {
		log.Printf("error creating clientset: %v\n", err)
		return nil, nil, false
	}

	log.Printf("success load client from default cluster config\n")
	apiClient = client
	return client, err, true
}

func GetKubernetesClientForTarget(apiServerURL string) (*kubernetes.Clientset, error, bool) {
	//load config from default config
	kubernetesClientConfig, err := k8srest.InClusterConfig()
	if err != nil {
		if config.Conf.Debug {
			log.Printf("error creating clientset: %v\n", err)
		}
		return nil, nil, false
	}

	kubernetesClientConfig.Host = apiServerURL
	if !config.Conf.KubeClientTlsVerify {
		kubernetesClientConfig.TLSClientConfig.Insecure = true
		kubernetesClientConfig.TLSClientConfig.CAFile = ""
		kubernetesClientConfig.TLSClientConfig.CAData = nil
	}
	//load client from default cluster config
	if config.Conf.Debug {
		log.Printf("creating clientset: %v\n", kubernetesClientConfig.Host)
	}
	client, err := kubernetes.NewForConfig(kubernetesClientConfig)
	if err != nil {
		log.Printf("error creating clientset: %v\n", err)
		return nil, nil, false
	}
	if config.Conf.Debug {
		log.Printf("success load client from default cluster config\n")
	}
	return client, err, true
}
