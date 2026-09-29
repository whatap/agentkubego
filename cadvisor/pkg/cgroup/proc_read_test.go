package cgroup

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestReadSystemCPUUsageContract(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		want          int64
		wantError     bool
	}{
		{name: "aggregate only", content: "cpu0 999 999\n" + collectionTestCPU + "cpu1 888 888\n", want: 3600},
		{name: "older kernel fields", content: "cpu 100 200\n", want: 100 + 200},
		{name: "empty file"},
		{name: "missing aggregate", content: "cpu0 100 200\n"},
		{name: "ignore unrelated oversized row", content: collectionTestCPU + "intr " + strings.Repeat("1 ", bufio.MaxScanTokenSize), want: 3600},
		{name: "read failure", content: strings.Repeat("x", bufio.MaxScanTokenSize), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prefix := t.TempDir()
			writeCollectionFile(t, prefix, "proc/stat", tc.content)
			got, err := readSystemCPUUsage(prefix)
			if (err != nil) != tc.wantError || got != tc.want {
				t.Fatalf("usage=%d error=%v; want usage=%d error=%v", got, err, tc.want, tc.wantError)
			}
		})
	}
	if _, err := readSystemCPUUsage(t.TempDir()); !os.IsNotExist(err) {
		t.Fatalf("missing proc/stat error=%v", err)
	}
}

func TestReadContainerNetworkStatsReportsReadError(t *testing.T) {
	prefix := t.TempDir()
	writeCollectionFile(t, prefix, "proc/42/net/dev", networkTestHeader+strings.Repeat("x", bufio.MaxScanTokenSize))
	if _, err := readContainerNetworkStats(prefix, 42); !errors.Is(err, bufio.ErrTooLong) {
		t.Fatalf("scanner read failure was swallowed: %v", err)
	}
}
