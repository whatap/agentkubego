package router

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/whatap/kube/cadvisor/handler"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	"github.com/whatap/kube/tools/util/logutil"
	"net/http"
)

func Route() {
	/*
		init router
	*/
	var port string
	port = whatap_config.GetConfig().Port
	r := mux.NewRouter()
	r.HandleFunc("/health", handler.HealthHandler)
	r.HandleFunc("/debug/goroutine", handler.DebugGoroutineHandler)
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
