package router

import (
	. "github.com/whatap/kube/controlplane/router/internal"
	"github.com/whatap/kube/controlplane/scraper/etcd"
	"net/http"
)

func GetEtcdServerHasLeader(w http.ResponseWriter, r *http.Request) {
	metrics := etcd.GetCache("etcd_server_has_leader")
	var dataArr []EtcdServerHasLeader
	for _, m := range metrics {
		label := m.GetLabel()
		data := EtcdServerHasLeader{}
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
	WriteToJson(w, dataArr)
}

func GetEtcdServerProposalsCommittedTotal(w http.ResponseWriter, r *http.Request) {
	metrics := etcd.GetCache("etcd_server_proposals_committed_total")
	var dataArr []EtcdServerProposalsCommittedTotal
	for _, m := range metrics {
		label := m.GetLabel()
		data := EtcdServerProposalsCommittedTotal{}
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
	WriteToJson(w, dataArr)
}

func GetEtcdServerProposalsAppliedTotal(w http.ResponseWriter, r *http.Request) {
	metrics := etcd.GetCache("etcd_server_proposals_applied_total")
	var dataArr []EtcdServerProposalsAppliedTotal
	for _, m := range metrics {
		label := m.GetLabel()
		data := EtcdServerProposalsAppliedTotal{}
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
	WriteToJson(w, dataArr)
}

func GetEtcdServerLeaderChangesSeenTotal(w http.ResponseWriter, r *http.Request) {
	metrics := etcd.GetCache("etcd_server_leader_changes_seen_total")
	var dataArr []EtcdServerLeaderChangesSeenTotal
	for _, m := range metrics {
		label := m.GetLabel()
		data := EtcdServerLeaderChangesSeenTotal{}
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "Instance":
				data.Instance = value
			}
		}
		data.Counter = m.GetCounter().GetValue()
		dataArr = append(dataArr, data)
	}
	WriteToJson(w, dataArr)
}

type EtcdServerLeaderChangesSeenTotal struct {
	Counter  float64 `json:"counter"`
	Instance string  `json:"Instance"`
}

type EtcdServerHasLeader struct {
	Gauge    float64 `json:"gauge"`
	Instance string  `json:"Instance"`
}

type EtcdServerProposalsCommittedTotal struct {
	Gauge    float64 `json:"gauge"`
	Instance string  `json:"Instance"`
}

type EtcdServerProposalsAppliedTotal struct {
	Gauge    float64 `json:"gauge"`
	Instance string  `json:"Instance"`
}
