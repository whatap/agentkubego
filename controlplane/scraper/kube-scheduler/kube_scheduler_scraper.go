package kube_scheduler

import (
	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/whatap/kube/controlplane/pkg/config"
	"github.com/whatap/kube/controlplane/scraper/pkg/client"
	"github.com/whatap/kube/controlplane/scraper/pkg/token"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

func Do() {
	go StartTrackingSchedulerPod(1 * time.Minute)
	go scrap()

}

func scrap() {

	collectMetrics()

	ticker := time.NewTicker(time.Second * time.Duration(config.Conf.Cycle))

	defer ticker.Stop()

	for range ticker.C {
		collectMetrics()
	}

}

func collectMetrics() {
	ips := GetSchedulerPodIps()

	var wg sync.WaitGroup
	dataChan := make(chan MetricsData, len(ips))

	for _, ip := range ips {
		wg.Add(1)
		ip = "https://" + ip + ":" + config.Conf.KubeSchedulerPort + config.Conf.KubeSchedulerMetricsEndpoint
		go fetchMetrics(ip, &wg, dataChan)
	}
	go func() {
		wg.Wait()
		close(dataChan)
	}()
	mergeMetricsData(dataChan)
}

func fetchMetrics(url string, wg *sync.WaitGroup, dataChan chan<- MetricsData) {
	defer wg.Done()
	schedulerClient := client.GetSchedulerClient()
	token := token.GetServiceAccountTokenFromSecrets()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Cannot create request: %v\n", err)
		return
	}
	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := schedulerClient.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v\n", err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
	}

	var parser expfmt.TextParser
	families, err := parser.TextToMetricFamilies(strings.NewReader(string(responseBody)))

	var filteredMetricFamily = make(map[string]*ioprometheusclient.MetricFamily)
	for _, collectMetricsName := range collectMetricsNames {
		filteredMetricFamily[collectMetricsName] = families[collectMetricsName]
	}

	if err != nil {
		log.Printf("Error parsing response body: %v, %s", err, string(responseBody))
	}
	dataChan <- MetricsData{
		url,
		filteredMetricFamily,
	}
}

var collectMetricsNames []string = []string{
	"scheduler_pending_pods",
	"scheduler_preemption_attempts_total",
	"scheduler_preemption_victims"}

func mergeMetricsData(dataChan <-chan MetricsData) {
	allMetrics := make([]MetricsData, 0)
	for metricsData := range dataChan {
		allMetrics = append(allMetrics, metricsData)
	}
	rawMetricsCacheTmp := make(map[string]*ioprometheusclient.MetricFamily)
	for _, metricsData := range allMetrics {
		for name, metricFamily := range metricsData.Metrics {
			// add the 'instance' label to distinguish between sources
			updateMetricWithLabel(metricFamily, "instance", metricsData.SchedulerUrl)

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
}

type MetricsData struct {
	SchedulerUrl string
	Metrics      map[string]*ioprometheusclient.MetricFamily
}
