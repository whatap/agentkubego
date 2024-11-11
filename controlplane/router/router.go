package router

import (
	"github.com/gorilla/mux"
	"github.com/whatap/kube/controlplane/pkg/config"
	"log"
	"net/http"
)

func Route() {
	/*
		init router
	*/
	var router *mux.Router
	if config.Conf.CollectKubeApiserverMonitoringEnabled || config.Conf.CollectEtcdMonitoringEnabled || config.Conf.CollectKubeSchedulerMonitoringEnabled {
		router = mux.NewRouter()
	} else {
		return
	}

	/*
		kube-apiserver router
	*/
	if config.Conf.CollectKubeApiserverMonitoringEnabled {
		router.HandleFunc("/apiserver-request-duration-seconds", GetApiserverRequestDurationSeconds).Methods("GET")
		router.HandleFunc("/apiserver-request-total", GetApiserverRequestTotal).Methods("GET")
		router.HandleFunc("/apiserver-current-inflight-requests", GetApiserverCurrentInflightRequest).Methods("GET")
		router.HandleFunc("/apiserver-audit-level-total", GetApiserverAuditLevelTotal).Methods("GET")
		router.HandleFunc("/go-goroutines", GetGoGoroutines).Methods("GET")
		router.HandleFunc("/go-threads", GetGoThreads).Methods("GET")
		router.HandleFunc("/etcd-request-duration-seconds", GetEtcdRequestDurationSeconds).Methods("GET")
	}

	/*
		etcd router
	*/
	if config.Conf.CollectEtcdMonitoringEnabled {
		router.HandleFunc("/etcd-server-has-leader", GetEtcdServerHasLeader).Methods("GET")
		router.HandleFunc("/etcd-server-leader-changes-seen-total", GetEtcdServerLeaderChangesSeenTotal).Methods("GET")
		router.HandleFunc("/etcd-server-proposals-committed-total", GetEtcdServerProposalsCommittedTotal).Methods("GET")
		router.HandleFunc("/etcd-server-proposals-applied-total", GetEtcdServerProposalsAppliedTotal).Methods("GET")
	}

	/*
		scheduler router
	*/
	if config.Conf.CollectKubeSchedulerMonitoringEnabled {
		//router.HandleFunc("/etcd-server-has-leader", GetEtcdServerHasLeader).Methods("GET")
	}

	/*
		control plane helper http server
	*/
	if config.Conf.CollectKubeApiserverMonitoringEnabled || config.Conf.CollectEtcdMonitoringEnabled || config.Conf.CollectKubeSchedulerMonitoringEnabled {
		err := http.ListenAndServe(":"+config.Conf.Port, httpHandler(router))
		if err != nil {
			log.Println("http server error", err)
			return
		}
	}
}

func httpHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.Conf.Debug {
			log.Println(r.RemoteAddr, " ", r.Proto, " ", r.Method, " ", r.URL)
		}
		handler.ServeHTTP(w, r)
	})
}
