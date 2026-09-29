package osinfo

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProcessStatusFixture(t testing.TB, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestReadProcessStatusSkipsNonTargetStatus(t *testing.T) {
	pidDir := t.TempDir()
	writeProcessStatusFixture(t, filepath.Join(pidDir, "comm"), "unrelated\n")
	// A status read would fail even as root; do not rely on chmod permissions.
	if err := os.Mkdir(filepath.Join(pidDir, "status"), 0700); err != nil {
		t.Fatal(err)
	}
	got, err := readProcessStatus(pidDir, []string{"kubelet"})
	if err != nil || got != nil {
		t.Fatalf("non-target must skip status: got %q, err %v", got, err)
	}
}

func TestReadProcessStatusPreservesBytes(t *testing.T) {
	for _, name := range []string{"kubelet", "kube-controller", " name "} {
		t.Run(name, func(t *testing.T) {
			pidDir := t.TempDir()
			writeProcessStatusFixture(t, filepath.Join(pidDir, "comm"), name+"\n")
			want := []byte("Name:	" + name + "\nUid:	1000	1000	1000	1000\nVmSize:	 100 kB\n")
			writeProcessStatusFixture(t, filepath.Join(pidDir, "status"), string(want))
			got, err := readProcessStatus(pidDir, []string{name})
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("status bytes changed: got %q, err %v", got, err)
			}
		})
	}
}

func TestReadProcessStatusFallback(t *testing.T) {
	cases := []struct{ name, comm string }{
		{"missing", ""}, {"read-error", ""}, {"empty", ""},
		{"empty-name", "\n"}, {"incomplete", "other"},
		{"escaped-newline", "other\nname\n"}, {"escaped-backslash", "other\\name\n"},
		{"tab", "other	name\n"}, {"nul", "other\x00name\n"},
		{"oversized", strings.Repeat("x", 16) + "\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pidDir := t.TempDir()
			commPath := filepath.Join(pidDir, "comm")
			switch tc.name {
			case "missing":
			case "read-error":
				if err := os.Mkdir(commPath, 0700); err != nil {
					t.Fatal(err)
				}
			default:
				writeProcessStatusFixture(t, commPath, tc.comm)
			}
			want := "Name:	kubelet\nState:	S (sleeping)\n"
			writeProcessStatusFixture(t, filepath.Join(pidDir, "status"), want)
			got, err := readProcessStatus(pidDir, []string{"kubelet"})
			if err != nil || string(got) != want {
				t.Fatalf("uncertain comm must fall back: got %q, err %v", got, err)
			}
		})
	}
}

func TestReadProcessStatusFreshRequests(t *testing.T) {
	pidDir := t.TempDir()
	commPath, statusPath := filepath.Join(pidDir, "comm"), filepath.Join(pidDir, "status")
	writeProcessStatusFixture(t, commPath, "kubelet\n")
	writeProcessStatusFixture(t, statusPath, "Name:	kubelet\nThreads:	1\n")
	check := func(targets []string, want string) {
		t.Helper()
		got, err := readProcessStatus(pidDir, targets)
		if err != nil || string(got) != want {
			t.Fatalf("got %q, err %v; want %q", got, err, want)
		}
	}
	check([]string{"kubelet"}, "Name:	kubelet\nThreads:	1\n")
	check([]string{"containerd"}, "")
	check(nil, "")
	check([]string{}, "")
	writeProcessStatusFixture(t, statusPath, "Name:	kubelet\nThreads:	2\n")
	check([]string{"kubelet"}, "Name:	kubelet\nThreads:	2\n")
	writeProcessStatusFixture(t, commPath, "containerd\n")
	writeProcessStatusFixture(t, statusPath, "Name:	containerd\n")
	check([]string{"kubelet"}, "")
	check([]string{"containerd"}, "Name:	containerd\n")
	// A matching comm never replaces status Name validation in the caller.
	writeProcessStatusFixture(t, statusPath, "Name:	renamed\n")
	check([]string{"containerd"}, "Name:	renamed\n")
	if err := os.Remove(statusPath); err != nil {
		t.Fatal(err)
	}
	if _, err := readProcessStatus(pidDir, []string{"containerd"}); !os.IsNotExist(err) {
		t.Fatalf("deleted status error lost: %v", err)
	}
	if err := os.Mkdir(statusPath, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := readProcessStatus(pidDir, []string{"containerd"}); err == nil {
		t.Fatal("status read error lost")
	}
	if err := os.RemoveAll(pidDir); err != nil {
		t.Fatal(err)
	}
	if _, err := readProcessStatus(pidDir, []string{"containerd"}); !os.IsNotExist(err) {
		t.Fatalf("deleted PID error lost: %v", err)
	}
}

func BenchmarkReadProcessStatus(b *testing.B) {
	for _, matched := range []bool{false, true} {
		label := "non-target"
		name := "unrelated"
		if matched {
			label, name = "target", "kubelet"
		}
		b.Run(label, func(b *testing.B) {
			pidDir := b.TempDir()
			writeProcessStatusFixture(b, filepath.Join(pidDir, "comm"), name+"\n")
			writeProcessStatusFixture(b, filepath.Join(pidDir, "status"), "Name:	"+name+"\n"+strings.Repeat("VmSize:	100 kB\n", 100))
			b.Run("legacy-status", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if _, err := os.ReadFile(filepath.Join(pidDir, "status")); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("comm-prefilter", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if _, err := readProcessStatus(pidDir, []string{"kubelet"}); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
