package router

import (
	"net/http"
	"net/http/httptest"
	netpprof "net/http/pprof"
	"testing"

	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
)

func TestPprofGuardDisabled(t *testing.T) {
	whatap_config.GetConfig().DebugPprofEnabled = false

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/heap", nil)
	rec := httptest.NewRecorder()
	pprofGuard(netpprof.Index)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("debug_pprof_enabled=false 인데 %d 응답, 404 여야 함", rec.Code)
	}
}

func TestPprofGuardEnabled(t *testing.T) {
	whatap_config.GetConfig().DebugPprofEnabled = true
	defer func() { whatap_config.GetConfig().DebugPprofEnabled = false }()

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/heap", nil)
	rec := httptest.NewRecorder()
	pprofGuard(netpprof.Index)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("debug_pprof_enabled=true 인데 %d 응답, 200 이어야 함", rec.Code)
	}
}
