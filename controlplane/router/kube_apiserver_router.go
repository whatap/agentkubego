package router

import (
	"github.com/whatap/kube/controlplane/pkg/config"
	. "github.com/whatap/kube/controlplane/router/internal"
	"github.com/whatap/kube/controlplane/scraper/kube-apiserver"
	"log"
	"net/http"
	"strconv"
)

func GetApiserverRequestDurationSeconds(w http.ResponseWriter, r *http.Request) {
	metrics := kube_apiserver.GetCache("apiserver_request_duration_seconds")

	var buckets []ApiserverRequestDurationSecondsBucket
	var count []ApiserverRequestDurationSecondsCount
	var sum []ApiserverRequestDurationSecondsSum

	for _, m := range metrics {
		label := m.GetLabel()
		labelMap := make(map[string]string)
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "component":
				labelMap["component"] = value
			case "dryRun":
				labelMap["dryRun"] = value
			case "group":
				labelMap["group"] = value
			case "resource":
				labelMap["resource"] = value
			case "scope":
				labelMap["scope"] = value
			case "subresource":
				labelMap["subresource"] = value
			case "verb":
				labelMap["verb"] = value
			case "version":
				labelMap["version"] = value
			case "Instance":
				labelMap["Instance"] = value
			}
		}

		secondsCount := ApiserverRequestDurationSecondsCount{
			labelMap["component"],
			labelMap["dryRun"],
			labelMap["group"],
			labelMap["resource"],
			labelMap["scope"],
			labelMap["subresource"],
			labelMap["verb"],
			labelMap["version"],
			labelMap["Instance"],
			m.GetHistogram().GetSampleCount()}
		count = append(count, secondsCount)

		secondsSum := ApiserverRequestDurationSecondsSum{
			labelMap["component"],
			labelMap["dryRun"],
			labelMap["group"],
			labelMap["resource"],
			labelMap["scope"],
			labelMap["subresource"],
			labelMap["verb"],
			labelMap["version"],
			labelMap["Instance"],
			m.GetHistogram().GetSampleSum()}
		sum = append(sum, secondsSum)

		bucket := m.GetHistogram().GetBucket()
		for _, histogram := range bucket {
			data := ApiserverRequestDurationSecondsBucket{}
			data.Component = labelMap["component"]
			data.DryRun = labelMap["dryRun"]
			data.Group = labelMap["group"]
			data.Resource = labelMap["resource"]
			data.Scope = labelMap["scope"]
			data.Subresource = labelMap["subresource"]
			data.Verb = labelMap["verb"]
			data.Version = labelMap["version"]
			data.Instance = labelMap["Instance"]
			data.Le = strconv.FormatFloat(histogram.GetUpperBound(), 'f', -1, 64)
			data.CumulativeCount = histogram.GetCumulativeCount()
			buckets = append(buckets, data)
		}
	}

	var data ApiserverRequestDurationSeconds
	data.Buckets = buckets
	data.Count = count
	data.Sum = sum

	if config.Conf.Debug {
		log.Println("apiserver_request_duration_seconds metrics data arr", data)
	}

	WriteToJson(w, data)
}

func GetApiserverRequestTotal(w http.ResponseWriter, r *http.Request) {
	metrics := kube_apiserver.GetCache("apiserver_request_total")
	var dataArr []ApiServerRequestTotal
	for _, m := range metrics {
		label := m.GetLabel()
		data := ApiServerRequestTotal{}
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "code":
				data.Code = value
			case "component":
				data.Component = value
			case "dryRun":
				data.DryRun = value
			case "group":
				data.Group = value
			case "resource":
				data.Resource = value
			case "scope":
				data.Scope = value
			case "subresource":
				data.Subresource = value
			case "verb":
				data.Verb = value
			case "version":
				data.Version = value
			case "Instance":
				data.Instance = value
			}
		}
		data.Counter = m.GetCounter().GetValue()
		dataArr = append(dataArr, data)
	}
	if config.Conf.Debug {
		log.Println("apiserver_request_total metrics data arr", dataArr)
	}
	WriteToJson(w, dataArr)
}

func GetApiserverCurrentInflightRequest(w http.ResponseWriter, r *http.Request) {
	metrics := kube_apiserver.GetCache("apiserver_current_inflight_requests")
	var dataArr []ApiserverCurrentInflightRequests
	for _, m := range metrics {
		label := m.GetLabel()
		data := ApiserverCurrentInflightRequests{}
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "request_kind":
				data.RequestKind = value
			case "Instance":
				data.Instance = value
			}
		}
		data.Gauge = m.GetGauge().GetValue()
		dataArr = append(dataArr, data)
	}
	if config.Conf.Debug {
		log.Println("apiserver_current_inflight_requests metrics data arr", dataArr)
	}
	WriteToJson(w, dataArr)
}

func GetApiserverAuditLevelTotal(w http.ResponseWriter, r *http.Request) {
	metrics := kube_apiserver.GetCache("apiserver_audit_level_total")
	var dataArr []ApiserverAuditLevelTotal
	for _, m := range metrics {
		label := m.GetLabel()
		data := ApiserverAuditLevelTotal{}

		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "level":
				data.Level = value
			case "Instance":
				data.Instance = value
			}
		}
		data.Counter = m.GetCounter().GetValue()
		dataArr = append(dataArr, data)
	}
	if config.Conf.Debug {
		log.Println("apiserver_audit_level_total metrics data arr", dataArr)
	}
	WriteToJson(w, dataArr)
}

func GetGoGoroutines(w http.ResponseWriter, r *http.Request) {
	metrics := kube_apiserver.GetCache("go_goroutines")
	var dataArr []GoGoroutines
	for _, m := range metrics {
		label := m.GetLabel()
		data := GoGoroutines{}
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "Instance":
				data.Instance = value
			}
		}
		data.Gauge = m.GetGauge().GetValue()
		dataArr = append(dataArr, data)
	}
	if config.Conf.Debug {
		log.Println("go_goroutines metrics data arr", dataArr)
	}
	WriteToJson(w, dataArr)
}

func GetGoThreads(w http.ResponseWriter, r *http.Request) {
	metrics := kube_apiserver.GetCache("go_threads")
	var dataArr []GoThreads
	for _, m := range metrics {
		label := m.GetLabel()
		data := GoThreads{}
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "Instance":
				data.Instance = value
			}
		}
		data.Gauge = m.GetGauge().GetValue()
		dataArr = append(dataArr, data)
	}
	if config.Conf.Debug {
		log.Println("go_threads metrics data arr", dataArr)
	}
	WriteToJson(w, dataArr)
}

func GetEtcdRequestDurationSeconds(w http.ResponseWriter, r *http.Request) {
	metrics := kube_apiserver.GetCache("etcd_request_duration_seconds")

	var buckets []EtcdRequestDurationSecondsBucket
	var count []EtcdRequestDurationSecondsCount
	var sum []EtcdRequestDurationSecondsSum

	for _, m := range metrics {
		label := m.GetLabel()
		labelMap := make(map[string]string)
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "operation":
				labelMap["operation"] = value
			case "type":
				labelMap["type"] = value
			case "Instance":
				labelMap["Instance"] = value
			}
		}
		sampleCount := EtcdRequestDurationSecondsCount{
			labelMap["operation"],
			labelMap["type"],
			labelMap["Instance"],
			m.GetHistogram().GetSampleCount()}
		count = append(count, sampleCount)

		secondsSum := EtcdRequestDurationSecondsSum{
			labelMap["operation"],
			labelMap["type"],
			labelMap["Instance"],
			m.GetHistogram().GetSampleSum()}
		sum = append(sum, secondsSum)

		bucket := m.GetHistogram().GetBucket()
		for _, histogram := range bucket {
			data := EtcdRequestDurationSecondsBucket{}
			data.Operation = labelMap["operation"]
			data.Type = labelMap["type"]
			data.Instance = labelMap["Instance"]
			data.Le = strconv.FormatFloat(histogram.GetUpperBound(), 'f', -1, 64)
			data.CumulativeCount = histogram.GetCumulativeCount()
			buckets = append(buckets, data)
		}
	}

	var data EtcdRequestDurationSeconds
	data.Buckets = buckets
	data.Count = count
	data.Sum = sum

	if config.Conf.Debug {
		log.Println("etcd_request_duration_seconds metrics data arr", data)
	}

	WriteToJson(w, data)
}

/*
Counter of apiserver requests broken out for each verb, dry run value, group, version, resource, scope, component, and HTTP response code.
Stability Level:STABLE
Type: Counter
*/
type ApiServerRequestTotal struct {
	Code        string  `json:"code"`
	Component   string  `json:"component"`
	DryRun      string  `json:"dry_run"`
	Group       string  `json:"group"`
	Resource    string  `json:"resource"`
	Scope       string  `json:"scope"`
	Subresource string  `json:"subresource"`
	Verb        string  `json:"verb"`
	Version     string  `json:"version"`
	Instance    string  `json:"Instance"`
	Counter     float64 `json:"counter"`
}

/*
apiserver_request_duration_seconds
Response latency distribution in seconds for each verb, dry run value, group, version, resource, subresource, scope and component.
Stability Level:STABLE
Type: Histogram
*/
type ApiserverRequestDurationSeconds struct {
	Buckets []ApiserverRequestDurationSecondsBucket `json:"buckets"`
	Count   []ApiserverRequestDurationSecondsCount  `json:"count"`
	Sum     []ApiserverRequestDurationSecondsSum    `json:"sum"`
}

type ApiserverRequestDurationSecondsBucket struct {
	Component       string `json:"component"`
	DryRun          string `json:"dry_run"`
	Group           string `json:"group"`
	Resource        string `json:"resource"`
	Scope           string `json:"scope"`
	Subresource     string `json:"subresource"`
	Verb            string `json:"verb"`
	Version         string `json:"version"`
	Instance        string `json:"Instance"`
	Le              string `json:"le"`
	CumulativeCount uint64 `json:"cumulative_count"`
}

type ApiserverRequestDurationSecondsCount struct {
	Component   string `json:"component"`
	DryRun      string `json:"dry_run"`
	Group       string `json:"group"`
	Resource    string `json:"resource"`
	Scope       string `json:"scope"`
	Subresource string `json:"subresource"`
	Verb        string `json:"verb"`
	Version     string `json:"version"`
	Instance    string `json:"Instance"`
	SampleCount uint64 `json:"sample_count"`
}

type ApiserverRequestDurationSecondsSum struct {
	Component   string  `json:"component"`
	DryRun      string  `json:"dry_run"`
	Group       string  `json:"group"`
	Resource    string  `json:"resource"`
	Scope       string  `json:"scope"`
	Subresource string  `json:"subresource"`
	Verb        string  `json:"verb"`
	Version     string  `json:"version"`
	Instance    string  `json:"Instance"`
	SampleSum   float64 `json:"sample_sum"`
}

/*
Maximal number of currently used inflight request limit of this apiserver per request kind in last second.
Stability Level:STABLE
Type: Gauge
*/
type ApiserverCurrentInflightRequests struct {
	RequestKind string  `json:"request_kind"`
	Gauge       float64 `json:"gauge"`
	Instance    string  `json:"Instance"`
}

/*
Counter of policy levels for audit events (1 per request).
Stability Level:ALPHA
Type: Counter
*/
type ApiserverAuditLevelTotal struct {
	Level    string  `json:"level"`
	Instance string  `json:"Instance"`
	Counter  float64 `json:"counter"`
}

/*
Number of goroutines that currently exist.
Stability Level:?
Type: Gauge
*/
type GoGoroutines struct {
	Gauge    float64 `json:"gauge"`
	Instance string  `json:"Instance"`
}

/*
Number of OS threads created.
Stability Level:?
Type: Gauge
*/
type GoThreads struct {
	Gauge    float64 `json:"gauge"`
	Instance string  `json:"Instance"`
}

/*
Etcd request latency in seconds for each operation and object type.
Stability Level:ALPHA
Type: Histogram
*/
type EtcdRequestDurationSeconds struct {
	Buckets []EtcdRequestDurationSecondsBucket `json:"buckets"`
	Count   []EtcdRequestDurationSecondsCount  `json:"count"`
	Sum     []EtcdRequestDurationSecondsSum    `json:"sum"`
}

type EtcdRequestDurationSecondsBucket struct {
	Operation       string `json:"operation"`
	Type            string `json:"type"`
	Instance        string `json:"Instance"`
	Le              string `json:"le"`
	CumulativeCount uint64 `json:"cumulative_count"`
}

type EtcdRequestDurationSecondsCount struct {
	Operation   string `json:"operation"`
	Type        string `json:"type"`
	Instance    string `json:"Instance"`
	SampleCount uint64 `json:"sample_count"`
}

type EtcdRequestDurationSecondsSum struct {
	Operation string  `json:"operation"`
	Type      string  `json:"type"`
	Instance  string  `json:"Instance"`
	SampleSum float64 `json:"sample_sum"`
}
