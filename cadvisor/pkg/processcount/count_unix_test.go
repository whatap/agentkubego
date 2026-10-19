//go:build linux || darwin

package processcount

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestProcRootChangesDuringScan(t *testing.T) {
	for _, change := range []string{"removed", "replaced", "pid-exit"} {
		t.Run(change, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "proc")
			status(t, root, "20", "kubelet", "1")
			if err := os.Mkdir(filepath.Join(root, "10"), 0755); err != nil {
				t.Fatal(err)
			}
			fifo := filepath.Join(root, "10", "status")
			if err := syscall.Mkfifo(fifo, 0600); err != nil {
				t.Fatal(err)
			}
			type result struct {
				counts map[string]int
				err    error
			}
			done := make(chan result, 1)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				counts, err := Collect(root, []string{"kubelet"})
				done <- result{counts, err}
			}()
			type openResult struct {
				file *os.File
				err  error
			}
			opened := make(chan openResult, 1)
			go func() {
				file, err := os.OpenFile(fifo, os.O_WRONLY, 0600)
				opened <- openResult{file, err}
			}()
			var writer *os.File
			opening := true
			t.Cleanup(func() {
				if opening {
					// Release a blocked writer even if Collect never reaches the FIFO.
					reader, err := os.OpenFile(fifo, os.O_RDONLY|syscall.O_NONBLOCK, 0600)
					if err != nil {
						t.Error(err)
					} else {
						defer reader.Close()
					}
					select {
					case opened := <-opened:
						writer = opened.file
					case <-time.After(3 * time.Second):
						t.Error("FIFO writer did not stop during cleanup")
					}
				}
				closeScanFIFO(t, fifo, writer)
				select {
				case <-finished:
				case <-time.After(3 * time.Second):
					t.Error("Collect did not stop during cleanup")
				}
			})
			select {
			case opened := <-opened:
				opening = false
				writer = opened.file
				if opened.err != nil {
					t.Fatal(opened.err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("Collect did not open the FIFO")
			}
			// The FIFO rendezvous proves ReadDir completed before the change.
			if change == "pid-exit" {
				if err := os.RemoveAll(filepath.Join(root, "20")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Rename(root, filepath.Join(base, "detached-proc")); err != nil {
					t.Fatal(err)
				}
				if change == "replaced" {
					status(t, root, "20", "kubelet", "1")
				}
			}
			if _, err := writer.WriteString("Name:\tunrelated\nPPid:\t1\n"); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case got := <-done:
				if change == "pid-exit" {
					if got.err != nil || len(got.counts) != 1 || got.counts["kubelet"] != 0 {
						t.Fatalf("exited PID: counts=%v, err=%v; want kubelet=0 and no error", got.counts, got.err)
					}
				} else if got.err == nil || got.counts != nil {
					t.Fatalf("proc root %s: counts=%v, err=%v; want nil counts and an error", change, got.counts, got.err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("Collect did not finish")
			}
		})
	}
}

func closeScanFIFO(t *testing.T, fifo string, writer *os.File) {
	t.Helper()
	// Unlink before closing the writer so a late reader cannot block in open.
	if err := os.Remove(fifo); err != nil && !os.IsNotExist(err) {
		t.Error(err)
	}
	if writer != nil {
		writer.Close()
	}
}

func TestFIFOCleanupAllowsLateScan(t *testing.T) {
	root := t.TempDir()
	status(t, root, "20", "kubelet", "1")
	if err := os.Mkdir(filepath.Join(root, "10"), 0755); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(root, "10", "status")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	reader, err := os.OpenFile(fifo, os.O_RDONLY|syscall.O_NONBLOCK, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	writer, err := os.OpenFile(fifo, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	closeScanFIFO(t, fifo, writer)
	// Assert before entering Collect so a regression cannot strand a reader.
	if _, err := os.Stat(fifo); !os.IsNotExist(err) {
		t.Fatalf("cleanup left a FIFO that can block a late scan: %v", err)
	}
	counts, err := Collect(root, []string{"kubelet"})
	if err != nil || len(counts) != 1 || counts["kubelet"] != 1 {
		t.Fatalf("late scan: counts=%v, err=%v; want kubelet=1 and no error", counts, err)
	}
}
