package kube_apiserver

import (
	"github.com/prometheus/client_model/go"
	"github.com/whatap/kube/controlplane/pkg/config"
	"log"
)

var apiserverMetricsCache map[string]*io_prometheus_client.MetricFamily

func init() {
	if config.Conf.CollectKubeApiserverMonitoringEnabled {
		apiserverMetricsCache = make(map[string]*io_prometheus_client.MetricFamily)
	}
}

func GetCache(familyName string) []*io_prometheus_client.Metric {
	metricFamily := apiserverMetricsCache[familyName]
	if metricFamily == nil {
		if config.Conf.Debug {
			log.Println("can not get metrics from cache, family name ", familyName, " is nil")
		}
		return nil
	}
	return metricFamily.GetMetric()
}

func SetCache(rawMetric map[string]*io_prometheus_client.MetricFamily) {
	apiserverMetricsCache = rawMetric
}
