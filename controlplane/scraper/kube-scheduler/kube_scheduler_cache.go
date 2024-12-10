package kube_scheduler

import (
	io_prometheus_client "github.com/prometheus/client_model/go"
	"log"
	"sync"
)

var (
	currentSchedulerMetricsCache map[string]*io_prometheus_client.MetricFamily
	cacheMutex                   sync.RWMutex // 동시 접근을 위한 Mutex
)

func init() {
	currentSchedulerMetricsCache = make(map[string]*io_prometheus_client.MetricFamily)
}

// GetCache: 현재 캐시에서 MetricFamily를 가져옴
func GetCache(familyName string) []*io_prometheus_client.Metric {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	metricFamily := currentSchedulerMetricsCache[familyName]
	if metricFamily == nil {
		log.Println("Cannot get metrics from cache, family name", familyName, "is nil")
		return nil
	}
	return metricFamily.GetMetric()
}

// SetCache: 새로운 데이터를 받아 캐시를 갱신
func SetCache(rawMetric map[string]*io_prometheus_client.MetricFamily) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	currentSchedulerMetricsCache = rawMetric
}
