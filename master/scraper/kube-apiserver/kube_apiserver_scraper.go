package kube_apiserver

import (
	"context"
	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/whatap/kube/master/pkg/config"
	"k8s.io/client-go/kubernetes"
	"log"
	"strings"
	"sync"
	"time"
)

func Do() {
	// Run endpoint informer
	RunEndpointInformer()

	// collect metrics periodically(cycle=conf.Cycle)
	// collectMetrics 메소드를 주기적으로 실행하는 고루틴
	// collectMetrics 프로세스는 다음과 같음
	// 1. dicovery_kube.GetEndpointsByName : 엔드포인트 비동기적으로 업데이트
	// 2. go fetchMetricsFromPod : apiserver 파드 /metrics 데이터 병렬 요청
	// 3. mergeMetricsData : 요청받은 데이터 merge
	go collectMetricsPeriodically()
}

type MetricsData struct {
	ApiServerURL string
	Metrics      map[string]*ioprometheusclient.MetricFamily
}

func collectMetricsPeriodically() {
	collectMetrics()
	ticker := time.NewTicker(time.Second * time.Duration(config.Conf.Cycle))
	defer ticker.Stop()
	for range ticker.C {
		collectMetrics()
	}
}
func collectMetrics() {

	// 엔드포인트 가져오기
	simpleEndpointInfo, getENdpointByNameOk := GetEndpointsByName("kubernetes")
	if !getENdpointByNameOk {
		if config.Conf.Debug {
			log.Println("GetEndpointsByName(kubernetes)=nil")
		}
		return
	}
	urls := simpleEndpointInfo.Urls
	targetClients := simpleEndpointInfo.TargetClient
	//병렬처리 로직
	var wg sync.WaitGroup
	dataChan := make(chan MetricsData, len(urls))

	for _, apiServerURL := range urls {

		// 클라이언트 가져오기
		//kubeClientForTargetApiServerPod, err, done := client.G(apiServerURL)
		targetClient, getTargetClientOk := targetClients[apiServerURL]
		if !getTargetClientOk {
			if config.Conf.Debug {
				log.Printf("targetClients[%v]=nil \n", apiServerURL)
			}
			return
		}
		wg.Add(1)
		go fetchMetricsFromPod(targetClient, apiServerURL, &wg, dataChan)
	}
	go func() {
		wg.Wait()
		close(dataChan)
	}()
	mergeMetricsData(dataChan)
}

func fetchMetricsFromPod(clientForTarget *kubernetes.Clientset, apiServerURL string, wg *sync.WaitGroup, dataChan chan<- MetricsData) {
	defer wg.Done()
	bytes, err := clientForTarget.RESTClient().Get().AbsPath("/metrics").DoRaw(context.Background())
	if err != nil {
		if config.Conf.Debug {
			log.Printf("error getting metrics: %v\n", err)
		}
	}
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(strings.NewReader(string(bytes)))
	if err != nil {
		if config.Conf.Debug {
			log.Printf("error parsing metrics: %v\n", err)
		}
	}

	if config.Conf.Debug {
		log.Println("update raw metrics cache", "apiServerURL=", apiServerURL, "datetime=", time.Now().Unix())
		log.Println("update raw metrics cache", "apiServerURL=", apiServerURL, "data=", metricFamilies)
	}
	dataChan <- MetricsData{
		ApiServerURL: apiServerURL,
		Metrics:      metricFamilies,
	}
}
func mergeMetricsData(dataChan <-chan MetricsData) {
	allMetrics := make([]MetricsData, 0)
	for metricsData := range dataChan {
		allMetrics = append(allMetrics, metricsData)
	}
	rawMetricsCacheTmp := make(map[string]*ioprometheusclient.MetricFamily)
	for _, metricsData := range allMetrics {
		for name, metricFamily := range metricsData.Metrics {
			// add the 'instance' label to distinguish between sources
			updateMetricWithLabel(metricFamily, "instance", metricsData.ApiServerURL)

			// 캐시에 키 등록되어 있는 경우 기존의 배열에 append
			if existingMetricFamily, exists := rawMetricsCacheTmp[name]; exists {
				// Merge existing metrics with new ones
				for _, metric := range metricFamily.Metric {
					existingMetricFamily.Metric = append(existingMetricFamily.Metric, metric)
				}

				// 캐시에 키 등록되어 있지 않은 경우
			} else {
				rawMetricsCacheTmp[name] = metricFamily
			}
		}
	}
	SetCache(rawMetricsCacheTmp)
}

func updateMetricWithLabel(metricFamily *ioprometheusclient.MetricFamily, labelName string, labelValue string) {
	for _, metric := range metricFamily.Metric {
		label := &ioprometheusclient.LabelPair{
			Name:  &labelName,
			Value: &labelValue}
		metric.Label = append(metric.Label, label)
	}

	//for _, metric := range metricFamily.Metric {
	//	found := false
	//	for _, label := range metric.Label {
	//		if *label.Name == labelName {
	//			*label.Value = labelValue
	//			found = true
	//			break
	//		}
	//	}
	//	if !found {
	//		label := &io_prometheus_client.LabelPair{
	//			Name:  &labelName,
	//			Value: &labelValue}
	//		metric.Label = append(metric.Label, label)
	//	}
	//}
}
