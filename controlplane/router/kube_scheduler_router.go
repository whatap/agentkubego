package router

import (
	"github.com/whatap/kube/controlplane/router/internal"
	kube_scheduler "github.com/whatap/kube/controlplane/scraper/kube-scheduler"
	"net/http"
)

func GetMetrics(w http.ResponseWriter, r *http.Request) {
	var results = make(map[string]KubeSchedulerByInstance)
	GetSchedulerPendingPods(results)
	GetSchedulerPreemptionAttemptsTotal(results)
	GetSchedulerPreemptionVictims(results)
	var values []KubeSchedulerByInstance
	for _, result := range results {
		values = append(values, result)
	}
	internal.WriteToJson(w, values)
}

func GetSchedulerPreemptionAttemptsTotal(result map[string]KubeSchedulerByInstance) {
	cache := kube_scheduler.GetAttemptsTotalCache("scheduler_preemption_attempts_total")
	if cache != nil {
		for _, v := range cache {
			r := result[v.Instance]
			r.SchedulerPreemptionAttemptsTotal = v.SchedulerPreemptionAttemptsTotal
			result[v.Instance] = r
		}
	}
}

func GetSchedulerPreemptionVictims(result map[string]KubeSchedulerByInstance) {
	cache := kube_scheduler.GetVictims("scheduler_preemption_victims")
	if cache != nil {
		for _, v := range cache {
			r := result[v.Instance]
			r.SchedulerPreemptionVictimCount = v.SchedulerPreemptionVictims
			result[v.Instance] = r
		}
	}
}

func GetSchedulerPendingPods(result map[string]KubeSchedulerByInstance) {
	schedulerPendingPods := kube_scheduler.GetCache("scheduler_pending_pods")
	var temp = KubeSchedulerByInstance{}
	for _, pendingMetric := range schedulerPendingPods {
		label := pendingMetric.GetLabel()
		gauge := pendingMetric.GetGauge().GetValue()
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			if name == "queue" {
				if value == "active" {
					temp.SchedulerPendingPodsActive = gauge
				} else if value == "backoff" {
					temp.SchedulerPendingPodsBackoff = gauge
				} else if value == "gated" {
					temp.SchedulerPendingPodsGated = gauge
				} else if value == "unschedulable" {
					temp.SchedulerPendingPodsUnschedulable = gauge
				}
			}
			if name == "instance" {
				temp.Instance = value
			}
		}
		if temp.Instance != "" {
			result[temp.Instance] = KubeSchedulerByInstance{
				SchedulerPendingPodsActive:        temp.SchedulerPendingPodsActive,
				SchedulerPendingPodsBackoff:       temp.SchedulerPendingPodsBackoff,
				SchedulerPendingPodsGated:         temp.SchedulerPendingPodsGated,
				SchedulerPendingPodsUnschedulable: temp.SchedulerPendingPodsUnschedulable,
				Instance:                          temp.Instance,
			}
		}
	}
}

type KubeSchedulerByInstance struct {
	SchedulerPendingPodsActive        float64 `json:"pendingPodsActive"`
	SchedulerPendingPodsBackoff       float64 `json:"pendingPodsBackoff"`
	SchedulerPendingPodsGated         float64 `json:"pendingPodsGated"`
	SchedulerPendingPodsUnschedulable float64 `json:"pendingPodsUnschedulable"`
	SchedulerPreemptionAttemptsTotal  float64 `json:"preemptionAttemptsTotal"`
	SchedulerPreemptionVictimCount    uint64  `json:"preemptionVictimCount"`
	Instance                          string  `json:"instance"`
}
