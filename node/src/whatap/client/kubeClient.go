package client

import (
	whatap_config "github.com/whatap/kube/node/src/whatap/config"
	"github.com/whatap/kube/node/src/whatap/util/logutil"
	"k8s.io/client-go/kubernetes"
	k8srest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var apiClient *kubernetes.Clientset

func GetKubernetesClient() (*kubernetes.Clientset, error) {
	if apiClient != nil {
		return apiClient, nil
	}

	//load config from default config
	config, err := k8srest.InClusterConfig()
	if err != nil {
		//fmt.Printf("error getting out-of-cluster config: %v\n", err)
		kubeConfig := whatap_config.GetConfig().KubeConfigPath
		config, err = clientcmd.BuildConfigFromFlags(whatap_config.GetConfig().KubeMasterUrl, kubeConfig)
		if err != nil {
			logutil.Debugf("whatap-node-helper", "error getting out-of-cluster config: %v\n", err)
			return nil, nil
		}
	}

	//load client from default cluster config
	client, err := kubernetes.NewForConfig(config)
	if client == nil {
		logutil.Debugf("whatap-node-helper", "NewForConfigError: %v\n", err)
		return nil, nil
	}
	if err != nil {
		logutil.Debugf("whatap-node-helper", "error creating clientset: %v\n", err)
		return nil, nil
	}
	logutil.Infoln("whatap-node-helper", "success load client")
	apiClient = client
	err = nil
	return client, err
}

func GetKubernetesClientForTarget(apiServerURL string) (*kubernetes.Clientset, error, bool) {
	//load config from default config
	config, err := k8srest.InClusterConfig()
	if err != nil {
		if whatap_config.GetConfig().Debug {
			logutil.Debugf("whatap-node-helper", "error creating clientset: %v\n", err)
		}
		return nil, nil, false
	}

	config.Host = apiServerURL
	//load client from default cluster config
	if whatap_config.GetConfig().Debug {
		logutil.Debugf("whatap-node-helper", "creating clientset: %v\n", config.Host)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		logutil.Debugf("whatap-node-helper", "error creating clientset: %v\n", err)
		return nil, nil, false
	}
	if whatap_config.GetConfig().Debug {
		logutil.Debugf("whatap-node-helper", "success load client from default cluster config\n")
	}
	return client, err, true
}
