package etcd

import (
	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/whatap/kube/master/pkg/config"
	"github.com/whatap/kube/master/scraper/pkg/client"
	"io"
	"log"
	"strings"
	"sync"
	"time"
)

func Do() {

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
	hosts := config.Conf.EtcdHosts

	var wg sync.WaitGroup
	dataChan := make(chan MetricsData, len(hosts))

	for _, host := range hosts {
		wg.Add(1)
		host = "https://" + host + ":" + config.Conf.EtcdPort + config.Conf.EtcdMetricsEndpoint
		go fetchMetrics(host, &wg, dataChan)
	}
	go func() {
		wg.Wait()
		close(dataChan)
	}()
	mergeMetricsData(dataChan)
}

func fetchMetrics(url string, wg *sync.WaitGroup, dataChan chan<- MetricsData) {
	defer wg.Done()
	etcdClient := client.GetEtcdClient()
	resp, err := etcdClient.Get(url)
	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	var parser expfmt.TextParser
	families, err := parser.TextToMetricFamilies(strings.NewReader(string(responseBody)))
	if err != nil {
		log.Fatalf("Error parsing response body: %v, %s", err, string(responseBody))
	}
	dataChan <- MetricsData{
		url,
		families,
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
			updateMetricWithLabel(metricFamily, "instance", metricsData.EtcdUrl)

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
	EtcdUrl string
	Metrics map[string]*ioprometheusclient.MetricFamily
}
