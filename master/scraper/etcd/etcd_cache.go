package etcd

import (
	"github.com/prometheus/client_model/go"
	"github.com/whatap/kube/master/pkg/config"
	"log"
)

var etcdMetricsCache map[string]*io_prometheus_client.MetricFamily

func init() {
	if config.Conf.EtcdMonitoringEnabled {
		etcdMetricsCache = make(map[string]*io_prometheus_client.MetricFamily)
	}
}

func GetCache(familyName string) []*io_prometheus_client.Metric {
	metricFamily := etcdMetricsCache[familyName]
	if metricFamily == nil {
		if config.Conf.Debug {
			log.Println("cannot get metrics from cache, family name ", familyName, " is nil")
		}
		return nil
	}
	return metricFamily.GetMetric()
}

func SetCache(etcdMetrics map[string]*io_prometheus_client.MetricFamily) {
	etcdMetricsCache = etcdMetrics
}
