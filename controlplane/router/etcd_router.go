package router

import (
	. "github.com/whatap/kube/controlplane/router/internal"
	"github.com/whatap/kube/controlplane/scraper/etcd"
	"net/http"
)

func GetEtcdServerHasLeader(w http.ResponseWriter, r *http.Request) {
	metrics := etcd.GetCache("etcd_server_has_leader")
	// 메트릭 부재 시 nil 슬라이스가 JSON null 로 직렬화되는 것을 막고 빈 배열([])을 반환한다. (KAZAA-438)
	dataArr := make([]EtcdServerHasLeader, 0)
	for _, m := range metrics {
		label := m.GetLabel()
		data := EtcdServerHasLeader{}
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "instance":
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
	dataArr := make([]EtcdServerProposalsCommittedTotal, 0) // KAZAA-438: nil 대신 []
	for _, m := range metrics {
		label := m.GetLabel()
		data := EtcdServerProposalsCommittedTotal{}
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "instance":
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
	dataArr := make([]EtcdServerProposalsAppliedTotal, 0) // KAZAA-438: nil 대신 []
	for _, m := range metrics {
		label := m.GetLabel()
		data := EtcdServerProposalsAppliedTotal{}
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "instance":
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
	dataArr := make([]EtcdServerLeaderChangesSeenTotal, 0) // KAZAA-438: nil 대신 []
	for _, m := range metrics {
		label := m.GetLabel()
		data := EtcdServerLeaderChangesSeenTotal{}
		for _, l := range label {
			name := l.GetName()
			value := l.GetValue()
			switch name {
			case "instance":
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
	Instance string  `json:"instance"`
}

type EtcdServerHasLeader struct {
	Gauge    float64 `json:"gauge"`
	Instance string  `json:"instance"`
}

type EtcdServerProposalsCommittedTotal struct {
	Gauge    float64 `json:"gauge"`
	Instance string  `json:"instance"`
}

type EtcdServerProposalsAppliedTotal struct {
	Gauge    float64 `json:"gauge"`
	Instance string  `json:"instance"`
}
