//go:build linux

package cgroup

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func TestProcReadersCloseFilesWithoutGC(t *testing.T) {
	prefix := t.TempDir()
	writeCollectionFile(t, prefix, "proc/stat", collectionTestCPU)
	writeCollectionFile(t, prefix, "proc/42/net/dev", networkTestHeader+"eth0:1000 10 1 2 0 0 0 0 2000 20 3 4 0 0 0 0\n")
	previousGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previousGC)
	assertClosed := func() {
		t.Helper()
		entries, err := os.ReadDir("/proc/self/fd")
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
			if err == nil && strings.HasPrefix(target, prefix+"/") {
				t.Fatalf("reader left open without GC: %s", target)
			}
		}
	}
	for i := 0; i < 20; i++ {
		if _, err := readSystemCPUUsage(prefix); err != nil {
			t.Fatal(err)
		}
		if _, err := readContainerNetworkStats(prefix, 42); err != nil {
			t.Fatal(err)
		}
	}
	assertClosed()
	writeCollectionFile(t, prefix, "proc/stat", strings.Repeat("x", bufio.MaxScanTokenSize))
	writeCollectionFile(t, prefix, "proc/42/net/dev", strings.Repeat("x", bufio.MaxScanTokenSize))
	if _, err := readSystemCPUUsage(prefix); err == nil {
		t.Fatal("expected CPU scanner error")
	}
	if _, err := readContainerNetworkStats(prefix, 42); err == nil {
		t.Fatal("expected network scanner error")
	}
	assertClosed()
}
