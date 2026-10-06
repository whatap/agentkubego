package javaenv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/docker/docker/client"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func TestFromEnvOnlyExactJavaPath(t *testing.T) {
	cases := []struct {
		name string
		env  []string
		want string
	}{
		{"nil", nil, ""}, {"unrelated", []string{"TOKEN=private", "OTHER_WHATAP_JAVA_AGENT_PATH=wrong"}, ""},
		{"exact", []string{"TOKEN=private", "WHATAP_JAVA_AGENT_PATH=/dir=a b/whatap.agent.jar"}, "/dir=a b/whatap.agent.jar"},
		{"last", []string{"WHATAP_JAVA_AGENT_PATH=old", "WHATAP_JAVA_AGENT_PATH=new"}, "new"},
		{"empty", []string{"WHATAP_JAVA_AGENT_PATH=old", "WHATAP_JAVA_AGENT_PATH="}, ""},
		{"malformed", []string{"WHATAP_JAVA_AGENT_PATH"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fromEnv(tc.env); got != tc.want {
				t.Fatalf("got=%q want=%q", got, tc.want)
			}
		})
	}
}
func TestOCISpecEnvironmentAndNils(t *testing.T) {
	for _, spec := range []*specs.Spec{nil, {}, {Process: &specs.Process{}}} {
		if got := fromSpec(spec); got != "" {
			t.Fatalf("got=%q", got)
		}
	}
	spec := &specs.Spec{Process: &specs.Process{Args: []string{"-javaagent:/old/whatap.agent.jar"}, Env: []string{"WHATAP_JAVA_AGENT_PATH=/resolved/whatap.agent.jar"}}}
	if got := fromSpec(spec); got != "/resolved/whatap.agent.jar" {
		t.Fatalf("got=%q", got)
	}
}
func TestDockerResolvedEnvNotImageCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1.40/containers/container-id/json" {
			t.Errorf("path=%s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"Config": map[string]any{"Env": []string{"TOKEN=private", "WHATAP_JAVA_AGENT_PATH=/resolved/whatap.agent.jar"}, "Cmd": []string{"-javaagent:/wrong/whatap.agent.jar"}}})
	}))
	defer srv.Close()
	cli, err := client.NewClientWithOpts(client.WithHost(srv.URL), client.WithVersion("1.40"))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	got, err := inspectDocker(context.Background(), cli, "container-id")
	if err != nil || got != "/resolved/whatap.agent.jar" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}
func TestDockerNilConfigAndCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) }))
	defer srv.Close()
	cli, err := client.NewClientWithOpts(client.WithHost(srv.URL), client.WithVersion("1.40"))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	if got, err := inspectDocker(context.Background(), cli, "id"); got != "" || err != nil {
		t.Fatalf("got=%q err=%v", got, err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if _, err := inspectDocker(ctx, cli, "id"); err == nil {
		t.Fatal("expected cancelled request")
	}
}
func TestCrioResolvedEnvAlternateStorageAndRetry(t *testing.T) {
	root := t.TempDir()
	alt := t.TempDir()
	t.Setenv("overlay_config_path", alt)
	if _, err := inspectCrio(root, "id"); err == nil {
		t.Fatal("missing runtime config must be retryable error")
	}
	file := filepath.Join(alt, "id", "userdata", "config.json")
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(`{"process":{"env":["TOKEN=private","WHATAP_JAVA_AGENT_PATH=/startup/whatap.agent.jar"],"args":["-javaagent:/wrong.jar"]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := inspectCrio(root, "id")
	if err != nil || got != "/startup/whatap.agent.jar" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}
func TestInspectMissingIDAndUnsupportedRuntime(t *testing.T) {
	for _, tc := range []struct{ runtime, id string }{{"containerd", ""}, {"unknown", "id"}} {
		if _, err := Inspect(tc.runtime, tc.id); err == nil {
			t.Fatal("expected safe error")
		}
	}
}

func TestCrioRejectsFIFOWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	t.Setenv("overlay_config_path", "")
	path := filepath.Join(root, "var/lib/containers/storage/overlay-containers/id/userdata/config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := inspectCrio(root, "id"); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("non-regular runtime config accepted")
		}
	case <-time.After(100 * time.Millisecond):
		// Release the blocked reader before failing; never leave a worker behind.
		writer, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0600)
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()
		<-done
		t.Fatal("runtime lookup blocked opening a non-regular config")
	}
}
