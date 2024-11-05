package kube_apiserver

import (
	"fmt"
	"github.com/whatap/kube/master/scraper/pkg/client"
	"github.com/whatap/kube/master/tools/iputil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"log"
	"sync"
)

type SimpleEndpointInfo struct {
	Name         string
	Namespace    string
	Urls         []string
	TargetClient map[string]*kubernetes.Clientset
}

var (
	endPointsMap     map[string]SimpleEndpointInfo
	endpointsMapLock sync.RWMutex
)

func RunEndpointInformer() {
	kubeClient, err, done := client.GetKubernetesClient()
	if !done {
		log.Printf("InitializeInformer error getting client: %v\n", err)
	}
	// 인포머 팩토리 생성
	factory := informers.NewSharedInformerFactory(kubeClient, 0)
	endpointsInformer := factory.Core().V1().Endpoints().Informer()

	ch := make(chan struct{})

	// Workqueue 구성 및 생성
	//rateLimiter := workqueue.DefaultControllerRateLimiter()
	//queueConfig := workqueue.RateLimitingQueueConfig{
	//	Name: "Endpoints"}
	//workQueue := workqueue.NewRateLimitingQueueWithConfig(rateLimiter, queueConfig)

	//Endpoint 이벤트 핸들러 등록
	endpointsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			updateURLs(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			updateURLs(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			endpointsMapLock.Lock()
			defer endpointsMapLock.Unlock()
			endPointsMap = make(map[string]SimpleEndpointInfo) // Reset on delete
		},
	})
	go endpointsInformer.Run(ch)
	cache.WaitForCacheSync(ch, endpointsInformer.HasSynced)
}

// GetEndpoints 함수는 주어진 네임스페이스와 이름에 해당하는 엔드포인트를 반환
func updateURLs(obj interface{}) {
	endpoints, ok := obj.(*corev1.Endpoints)
	if !ok {
		log.Println("Object is not an Endpoints")
		return
	}
	epName := endpoints.GetName()
	epNamespace := endpoints.GetNamespace()
	var epUrls []string
	var epTargetClient = make(map[string]*kubernetes.Clientset)

	endpointsMapLock.Lock()
	defer endpointsMapLock.Unlock()
	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			for _, port := range subset.Ports {
				if port.Name == "https" {
					addressIp := address.IP
					portPort := port.Port
					if iputil.IsIPv6(addressIp) {
						addressIp = fmt.Sprintf("[%s]", addressIp)
					}
					epUrl := fmt.Sprintf("https://%s:%d", addressIp, portPort)
					epUrls = append(epUrls, epUrl)
					if epName == "kubernetes" {
						clientForTarget, err, done := client.GetKubernetesClientForTarget(epUrl)
						if !done || err != nil {
							log.Printf("error getting client: done=%v, err=%v \n", done, err)
							continue
						}
						epTargetClient[epUrl] = clientForTarget
					}
				}
			}
		}
	}
	if endPointsMap == nil {
		endPointsMap = make(map[string]SimpleEndpointInfo)
	}
	sep := SimpleEndpointInfo{}
	sep.Name = epName
	sep.Namespace = epNamespace
	sep.Urls = epUrls
	if epTargetClient != nil {
		sep.TargetClient = make(map[string]*kubernetes.Clientset)
		sep.TargetClient = epTargetClient
	}
	endPointsMap[epName] = sep
}

func GetEndpointsMap() map[string]SimpleEndpointInfo {
	endpointsMapLock.RLock()
	defer endpointsMapLock.RUnlock()
	return endPointsMap
}

func GetEndpointsByName(name string) (SimpleEndpointInfo, bool) {
	endpointsMapLock.RLock()
	defer endpointsMapLock.RUnlock()
	simpleEndpointInfo, ok := endPointsMap[name]
	return simpleEndpointInfo, ok // Return the found value and whether the key was present
}
