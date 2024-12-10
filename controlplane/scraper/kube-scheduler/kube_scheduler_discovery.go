package kube_scheduler

import (
	"context"
	"github.com/whatap/kube/controlplane/scraper/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"sync"
	"time"
)

var schedulerPodIpCache sync.Map

func StartTrackingSchedulerPod(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	reloadSchedulerInfo()
	var checkCounting = 0
	schedulerPodIpCache.Range(func(key, value interface{}) bool {
		checkCounting++
		return true
	})
	if checkCounting > 0 {
		log.Println("IP tracking of kube-scheduler was successful.")
	} else {
		log.Println("IP tracking of kube-scheduler was failed.")
	}

	// 이후 주기적으로 실행
	for range ticker.C {
		reloadSchedulerInfo()
	}
}

func reloadSchedulerInfo() {
	kubeClient, err, done := client.GetKubernetesClient()

	if !done {
		log.Printf("InitializeInformer error getting client: %v\n", err)
	}
	list, err := kubeClient.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "component=kube-scheduler",
	})
	if err != nil {
		log.Printf("Error getting kube-scheduler pods: %v\n", err)
	}

	schedulerPodIpCache = sync.Map{}
	for _, pod := range list.Items {
		name := pod.Name
		podIp := pod.Status.PodIP
		schedulerPodIpCache.Store(name, podIp)
	}
}

func GetSchedulerPodIps() []string {
	var result []string
	schedulerPodIpCache.Range(func(key, value interface{}) bool {
		podIp, ok := value.(string)
		if !ok {
			log.Printf("Unexpected value type in schedulerPodIpCache for key %v\n", key)
			return true
		}
		if podIp != "" {
			result = append(result, podIp)
		}
		return true
	})
	return result
}
