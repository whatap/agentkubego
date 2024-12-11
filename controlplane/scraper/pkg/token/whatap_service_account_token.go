package token

import (
	"context"
	"github.com/whatap/kube/controlplane/pkg/config"
	"github.com/whatap/kube/controlplane/scraper/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
)

var token string

func GetServiceAccountTokenFromSecrets() string {
	if len(token) == 0 {
		kubeClient, err, b := client.GetKubernetesClient()
		if !b {
			log.Printf("InitializeInformer error getting client: %v\n", err)
		}
		secret, err := kubeClient.CoreV1().Secrets("whatap-monitoring").Get(context.TODO(), "whatap-scheduler-monitoring-token", metav1.GetOptions{})
		if err != nil {
			log.Printf("InitializeInformer error getting secrets: %v\n", err)
		}
		tokenData, ok := secret.Data["token"]
		if !ok {
			log.Printf("Unable to retrieve token data from Secret: %v\n", err)
		}
		token = string(tokenData)

		if config.Conf.Debug {
			log.Println("Successfully acquired Whatap Scheduler Monitoring token.", token)
		}
	}
	return token
}
