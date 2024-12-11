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
	log.Println("why3")

	ticker := time.NewTicker(interval)
	log.Println("why4")

	defer ticker.Stop()
	log.Println("why5")

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
	log.Println("why6")
	kubeClient, err, done := client.GetKubernetesClient()
	log.Println("why7")

	if !done {
		log.Printf("InitializeInformer error getting client: %v\n", err)
	}
	log.Println("why8")

	list, err := kubeClient.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "component=kube-scheduler",
	})
	log.Println("why9")

	if err != nil {
		log.Printf("Error getting kube-scheduler pods: %v\n", err)
	}
	log.Println("why10")

	schedulerPodIpCache = sync.Map{}
	log.Println("why11")

	for _, pod := range list.Items {
		name := pod.Name
		podIp := pod.Status.PodIP
		schedulerPodIpCache.Store(name, podIp)
	}
	log.Println("why12")

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
