package cgroup

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

const blkioTestTimeout = 2 * time.Second

func writeCgroupReadFixture(t *testing.T, root, device, group, filename string) {
	t.Helper()
	dir := filepath.Join(root, "sys/fs/cgroup", device, group)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte("8:0 Read 123\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func awaitCgroupRead(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(blkioTestTimeout):
		t.Fatal("cgroup read did not complete")
	}
}

// Both calls use real files and the production parser, with distinct cgroups:
// blkcg_print_blkgs contends on a kernel lock shared across those cgroups.
func TestBlkioReadsDoNotOverlapAcrossContainers(t *testing.T) {
	root := t.TempDir()
	const filename = "blkio.throttle.io_service_bytes_recursive"
	writeCgroupReadFixture(t, root, "blkio", "first", filename)
	writeCgroupReadFixture(t, root, "blkio", "second", filename)
	firstEntered := make(chan struct{})
	secondEntered := make(chan struct{})
	release := make(chan struct{})
	firstDone, secondDone := make(chan error, 1), make(chan error, 1)
	defer func() {
		close(release)
		awaitCgroupRead(t, firstDone)
		awaitCgroupRead(t, secondDone)
	}()

	go func() {
		firstDone <- populateCgroupValues(root, "blkio", "first", filename, func(tokens []string) {
			close(firstEntered)
			<-release
		})
	}()
	select {
	case <-firstEntered:
	case <-time.After(blkioTestTimeout):
		t.Fatal("first reader did not enter")
	}
	go func() {
		secondDone <- populateCgroupValues(root, "blkio", "second", filename, func(tokens []string) {
			close(secondEntered)
		})
	}()
	select {
	case <-secondEntered:
		t.Fatal("overlapping blkio reads across containers: kernel lock contention is unbounded")
	case <-time.After(100 * time.Millisecond):
		// The second read must wait for the first file to finish and close.
	}
}

func TestBlkioReadRetainsTokensAndReleasesAfterOpenError(t *testing.T) {
	root := t.TempDir()
	const filename = "blkio.throttle.io_serviced_recursive"
	if err := populateCgroupValues(root, "blkio", "missing", filename, func([]string) {}); err == nil {
		t.Fatal("missing blkio file must return an error")
	}
	writeCgroupReadFixture(t, root, "blkio", "present", filename)
	done := make(chan error, 1)
	var got [][]string
	go func() {
		done <- populateCgroupValues(root, "blkio", "present", filename, func(tokens []string) {
			got = append(got, tokens)
		})
	}()
	awaitCgroupRead(t, done)
	if !reflect.DeepEqual(got, [][]string{{"8:0", "Read", "123"}}) {
		t.Fatalf("parsed blkio tokens changed: %v", got)
	}
}

func TestBlockedBlkioDoesNotBlockOtherControllers(t *testing.T) {
	root := t.TempDir()
	writeCgroupReadFixture(t, root, "blkio", "first", "blkio.throttle.io_serviced_recursive")
	writeCgroupReadFixture(t, root, "", "unified", "io.stat")
	entered, release := make(chan struct{}), make(chan struct{})
	blkioDone, unifiedDone := make(chan error, 1), make(chan error, 1)
	go func() {
		blkioDone <- populateCgroupValues(root, "blkio", "first", "blkio.throttle.io_serviced_recursive", func([]string) {
			close(entered)
			<-release
		})
	}()
	defer func() {
		close(release)
		awaitCgroupRead(t, blkioDone)
	}()
	select {
	case <-entered:
	case <-time.After(blkioTestTimeout):
		t.Fatal("blkio reader did not enter")
	}
	go func() {
		unifiedDone <- populateCgroupValues(root, "", "unified", "io.stat", func([]string) {})
	}()
	awaitCgroupRead(t, unifiedDone)
}
