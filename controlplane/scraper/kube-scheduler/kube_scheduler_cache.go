package kube_scheduler

import (
	io_prometheus_client "github.com/prometheus/client_model/go"
	"log"
	"sync"
)

var (
	prevSchedulerMetricsCache    map[string]*io_prometheus_client.MetricFamily
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

type Victims struct {
	SchedulerPreemptionVictimCount uint64 `json:"preemptionAttemptsTotal"`
	Instance                       string `json:"instance"`
}

type AttemptsTotal struct {
	SchedulerPreemptionAttemptsTotal float64 `json:"preemptionAttemptsTotal"`
	Instance                         string  `json:"instance"`
}

func GetVictims(familyName string) []Victims {
	if prevSchedulerMetricsCache[familyName] == nil {
		prevSchedulerMetricsCache = make(map[string]*io_prometheus_client.MetricFamily)
		prevSchedulerMetricsCache[familyName] = currentSchedulerMetricsCache[familyName]
		return nil
	} else {
		family := currentSchedulerMetricsCache[familyName]
		metric := family.GetMetric()
		var results []Victims
		//metrics
		for _, m := range metric {
			label := m.GetLabel()
			//label
			for _, l := range label {
				if l.GetName() == "instance" {
					instance := l.GetValue()
					currentValue := m.GetHistogram().GetSampleCount()

					metricFamily := prevSchedulerMetricsCache[familyName]
					for _, mm := range metricFamily.GetMetric() {
						getLabel := mm.GetLabel()
						for _, ll := range getLabel {
							if ll.GetName() == "instance" {
								prevInstance := ll.GetValue()
								prevValue := mm.GetHistogram().GetSampleCount()
								if instance == prevInstance {
									result := currentValue - prevValue
									if result < 0 {
										var box = Victims{
											Instance:                       instance,
											SchedulerPreemptionVictimCount: 0,
										}
										results = append(results, box)
									} else {
										var box = Victims{
											Instance:                       instance,
											SchedulerPreemptionVictimCount: result,
										}
										results = append(results, box)
									}
								}
							}
						}
					}
				}
			}
		}
		prevSchedulerMetricsCache[familyName] = currentSchedulerMetricsCache[familyName]
		return results
	}
}

func GetAttemptsTotalCache(familyName string) []AttemptsTotal {
	if prevSchedulerMetricsCache[familyName] == nil {
		prevSchedulerMetricsCache = make(map[string]*io_prometheus_client.MetricFamily)
		prevSchedulerMetricsCache[familyName] = currentSchedulerMetricsCache[familyName]
		return nil
	} else {
		family := currentSchedulerMetricsCache[familyName]
		metric := family.GetMetric()
		var results []AttemptsTotal
		//metrics
		for _, m := range metric {
			label := m.GetLabel()
			//label
			for _, l := range label {
				if l.GetName() == "instance" {
					instance := l.GetValue()
					currentValue := m.GetCounter().GetValue()

					metricFamily := prevSchedulerMetricsCache[familyName]
					for _, mm := range metricFamily.GetMetric() {
						getLabel := mm.GetLabel()
						for _, ll := range getLabel {
							if ll.GetName() == "instance" {
								prevInstance := ll.GetValue()
								prevValue := mm.GetCounter().GetValue()
								if instance == prevInstance {
									result := currentValue - prevValue
									if result < 0 {
										var box = AttemptsTotal{
											Instance:                         instance,
											SchedulerPreemptionAttemptsTotal: 0,
										}
										results = append(results, box)
									} else {
										var box = AttemptsTotal{
											Instance:                         instance,
											SchedulerPreemptionAttemptsTotal: result,
										}
										results = append(results, box)
									}
								}
							}
						}
					}
				}
			}
		}
		prevSchedulerMetricsCache[familyName] = currentSchedulerMetricsCache[familyName]
		return results
	}
}
