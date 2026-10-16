package handler

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
)

func TestProcessCountResponse(t *testing.T) {
	conf := whatap_config.GetConfig()
	oldPath, oldTargets := conf.HostPathPrefix, conf.CollectKubeNodeProcessMetricTargetList
	defer func() { conf.HostPathPrefix = oldPath; conf.CollectKubeNodeProcessMetricTargetList = oldTargets }()
	conf.HostPathPrefix = t.TempDir()
	conf.CollectKubeNodeProcessMetricTargetList = []string{"kubelet", "dockerd"}
	if err := os.MkdirAll(filepath.Join(conf.HostPathPrefix, "proc", "10"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conf.HostPathPrefix, "proc", "10", "status"), []byte("Name:\tkubelet\nPPid:\t1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/host/process-counts", nil)
	res := httptest.NewRecorder()
	GetHostProcessCountHandler(res, req)
	var counts map[string]int
	if err := json.Unmarshal(res.Body.Bytes(), &counts); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || counts["kubelet"] != 1 || counts["dockerd"] != 0 || len(counts) != 2 {
		t.Fatalf("unexpected response %d: %s", res.Code, res.Body.String())
	}
	if err := os.RemoveAll(filepath.Join(conf.HostPathPrefix, "proc")); err != nil {
		t.Fatal(err)
	}
	res = httptest.NewRecorder()
	GetHostProcessCountHandler(res, req)
	if res.Code != 503 {
		t.Fatalf("failed scan returned %d: %s", res.Code, res.Body.String())
	}
}
