package router

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/whatap/kube/cadvisor/handler"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	"github.com/whatap/kube/tools/util/logutil"
	"net/http"
	netpprof "net/http/pprof"
)

// debug_pprof_enabled(기본 false)가 켜진 경우에만 pprof 핸들러를 노출한다.
// 설정 파일은 3초 주기로 리로드되므로 재시작 없이 whatap.conf 로 켜고 끌 수 있다.
func pprofGuard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if !whatap_config.GetConfig().DebugPprofEnabled {
			http.NotFound(w, req)
			return
		}
		h(w, req)
	}
}

func Route() {
	/*
		init router
	*/
	var port string
	port = whatap_config.GetConfig().Port
	r := mux.NewRouter()
	r.HandleFunc("/health", handler.HealthHandler)
	r.HandleFunc("/debug/goroutine", handler.DebugGoroutineHandler)

	r.HandleFunc("/debug/pprof/cmdline", pprofGuard(netpprof.Cmdline))
	r.HandleFunc("/debug/pprof/profile", pprofGuard(netpprof.Profile))
	r.HandleFunc("/debug/pprof/symbol", pprofGuard(netpprof.Symbol))
	r.HandleFunc("/debug/pprof/trace", pprofGuard(netpprof.Trace))
	// heap/goroutine/allocs 등 이름 있는 프로파일은 Index가 경로에서 찾아 처리한다.
	r.PathPrefix("/debug/pprof/").HandlerFunc(pprofGuard(netpprof.Index))
	r.HandleFunc("/container", handler.GetAllContainerHandler)
	r.HandleFunc("/container/{containerid}", handler.GetContainerInspectHandler)
	r.HandleFunc("/container/{containerid}/stats", handler.GetContainerStatsHandler)

	r.HandleFunc("/container/{containerid}/logs", handler.GetContainerLogHandler).
		Queries("stdout", "{stdout}").
		Queries("stderr", "{stderr}").
		Queries("since", "{since}").
		Queries("until", "{until}").
		Queries("timestamps", "{timestamps}").
		Queries("tail", "{tail}")

	r.HandleFunc("/container/{containerid}/volumes", handler.GetContainerVolumeHandler)
	r.HandleFunc("/container/{containerid}/netstat", handler.GetContainerNetstatHandler)

	r.HandleFunc("/host/disks", handler.GetHostDiskHandler)
	r.HandleFunc("/host/processes", handler.GetHostProcessHandler)

	// fmt.Println(time.Now(), "trying to listen", fmt.Sprintf(":%d", *port))
	// loggedRouter := handlers.CombinedLoggingHandler(os.Stdout, r)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: r,
	}
	err := srv.ListenAndServe()
	if err != nil {
		logutil.Errorln("ListenAndServe", err)
		return
	}
}
