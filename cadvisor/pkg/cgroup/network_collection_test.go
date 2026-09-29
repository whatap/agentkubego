package cgroup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	whatap_model "github.com/whatap/kube/cadvisor/pkg/model"
)

const networkTestHeader = "Inter-| Receive | Transmit\n face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed\n"

func TestContainerNetworkStatsParsing(t *testing.T) {
	cases := []struct {
		name string
		rows string
		want whatap_model.ContainerNetworkStats
	}{
		{
			name: "multiple interfaces and excluded loopback",
			rows: "eth0: 1000 10 1 2 0 0 0 0 2000 20 3 4 0 0 0 0\n" +
				"lo: 9999 9999 99 99 0 0 0 0 9999 9999 99 99 0 0 0 0\n" +
				"net1: 100 1 2 3 0 0 0 0 200 2 4 5 0 0 0 0\n",
			want: whatap_model.ContainerNetworkStats{RxBytes: 1100, RxPackets: 11, RxErrors: 3, RxDropped: 5, TxBytes: 2200, TxPackets: 22, TxErrors: 7, TxDropped: 9},
		},
		{
			name: "counters directly after colon",
			rows: "eth0:1000 10 1 2 0 0 0 0 2000 20 3 4 0 0 0 0\n" +
				"lo:9999 9999 99 99 0 0 0 0 9999 9999 99 99 0 0 0 0\n",
			want: whatap_model.ContainerNetworkStats{RxBytes: 1000, RxPackets: 10, RxErrors: 1, RxDropped: 2, TxBytes: 2000, TxPackets: 20, TxErrors: 3, TxDropped: 4},
		},
		{name: "empty and truncated rows", rows: "\neth0: 100 10\nno-colon 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16\n"},
		{name: "headers only"},
	}
	for _, version := range []string{"v1", "v2"} {
		for _, tc := range cases {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				prefix := setupCollectionFixture(t, version)
				writeCollectionFile(t, prefix, "proc/42/net/dev", networkTestHeader+tc.rows)
				stat, err := collectFixtureStats(prefix, version)
				if err != nil {
					t.Fatal(err)
				}
				if stat.NetworkStats != tc.want {
					t.Fatalf("network stats=%+v, want %+v", stat.NetworkStats, tc.want)
				}
			})
		}
	}
}

func TestContainerStatsNetworkWireAndFreshness(t *testing.T) {
	conf := whatap_config.GetConfig()
	oldMode := conf.CgroupVersion
	t.Cleanup(func() { conf.CgroupVersion = oldMode })
	for _, tc := range []struct{ version, mode string }{{"v1", "legacy"}, {"v2", "unified"}} {
		t.Run(tc.version, func(t *testing.T) {
			conf.CgroupVersion = tc.mode
			prefix := setupCollectionFixture(t, tc.version)
			for _, value := range []struct {
				row  string
				want int64
			}{
				{"eth0: 1000 10 1 2 0 0 0 0 2000 20 3 4 0 0 0 0\n", 1000},
				{"eth0: 1100 11 1 2 0 0 0 0 2200 22 3 4 0 0 0 0\n", 1100},
			} {
				writeCollectionFile(t, prefix, "proc/42/net/dev", networkTestHeader+value.row)
				wire, err := GetContainerStatsEx(prefix, "id", "name", testCgroupParent, 2, 42, 4096)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Count(wire, `"network_stats"`) != 1 {
					t.Fatalf("network_stats must be serialized exactly once: %s", wire)
				}
				var decoded whatap_model.ContainerStat
				if err := json.Unmarshal([]byte(wire), &decoded); err != nil {
					t.Fatal(err)
				}
				if decoded.NetworkStats.RxBytes != value.want || decoded.ID != "id" || decoded.Name != "name" || decoded.RestartCount != 2 {
					t.Fatalf("wire contract/freshness changed: %+v", decoded)
				}
			}
			if err := os.Remove(filepath.Join(prefix, "proc/42/net/dev")); err != nil {
				t.Fatal(err)
			}
			if _, err := GetContainerStatsEx(prefix, "id", "name", testCgroupParent, 2, 42, 4096); !os.IsNotExist(err) {
				t.Fatalf("missing network source must remain an error, got %v", err)
			}
		})
	}
}
