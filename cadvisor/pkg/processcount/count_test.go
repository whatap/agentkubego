package processcount

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func status(t *testing.T, root, pid, name, ppid string) {
	t.Helper()
	dir := filepath.Join(root, pid)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "status"), []byte("Name:\t"+name+"\nPPid:\t"+ppid+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCountsIncludeAbsentTargetsAndExit(t *testing.T) {
	root := t.TempDir()
	status(t, root, "10", "kubelet", "1")
	status(t, root, "11", "kubelet", "1")
	status(t, root, "12", "kube-controller", "1")
	status(t, root, "13", "ignored", "1")
	status(t, root, "2", "kubelet", "0")
	status(t, root, "14", "kubelet", "2")
	targets := []string{"kubelet", "dockerd", "kube-controller", "kube-controller-manager", "kubelet", ""}
	want := map[string]int{"kubelet": 2, "dockerd": 0, "kube-controller": 1, "kube-controller-manager": 0}
	counts, err := Collect(root, targets)
	if err != nil || !reflect.DeepEqual(counts, want) {
		t.Fatalf("got %v, %v; want %v", counts, err, want)
	}
	if err := os.RemoveAll(filepath.Join(root, "10")); err != nil {
		t.Fatal(err)
	}
	counts, err = Collect(root, targets)
	if err != nil || counts["kubelet"] != 1 {
		t.Fatalf("expected one: %v, %v", counts, err)
	}
	if err := os.RemoveAll(filepath.Join(root, "11")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		counts, err = Collect(root, targets)
		if err != nil || counts["kubelet"] != 0 {
			t.Fatalf("expected persistent zero: %v, %v", counts, err)
		}
	}
	counts, err = Collect(root, []string{"dockerd"})
	if err != nil || !reflect.DeepEqual(counts, map[string]int{"dockerd": 0}) {
		t.Fatalf("target reload: %v, %v", counts, err)
	}
}

func TestScanFailureIsNotZero(t *testing.T) {
	counts, err := Collect(filepath.Join(t.TempDir(), "missing"), []string{"kubelet"})
	if err == nil || counts != nil {
		t.Fatalf("failed scan returned %v, %v", counts, err)
	}
	root := t.TempDir()
	status(t, root, "10", "", "1")
	counts, err = Collect(root, []string{"kubelet"})
	if err == nil || counts != nil {
		t.Fatalf("invalid status returned %v, %v", counts, err)
	}
}

func TestExitedDuringScanAndEmptyTargets(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "10"), 0755); err != nil {
		t.Fatal(err)
	}
	counts, err := Collect(root, []string{"kubelet"})
	if err != nil || counts["kubelet"] != 0 {
		t.Fatalf("exited process: %v, %v", counts, err)
	}
	counts, err = Collect(root, nil)
	if err != nil || counts == nil || len(counts) != 0 {
		t.Fatalf("empty targets: %v, %v", counts, err)
	}
}
