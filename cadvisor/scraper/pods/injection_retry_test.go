package pods

import (
	"errors"
	"testing"
	"time"

	"k8s.io/client-go/tools/cache"
)

type execRecorder struct {
	paths []string
	fail  func(call int) bool
}

func resetInjectionRetries() {
	injectionRetries.Range(func(key, _ interface{}) bool {
		injectionRetries.Delete(key)
		return true
	})
}

// stubInjection replaces the in-container exec and the clock. The returned clock pointer moves time.
func stubInjection(t *testing.T, fail func(call int) bool) (*execRecorder, *time.Time) {
	t.Helper()
	resetPodCaches()
	resetInjectionRetries()
	rec := &execRecorder{fail: fail}
	clock := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	origExec, origNow := execAgentCommand, nowFunc
	execAgentCommand = func(_, _, _, _, agentPath, _ string) (bool, error) {
		rec.paths = append(rec.paths, agentPath)
		if rec.fail(len(rec.paths)) {
			return false, errors.New("stat " + agentPath + ": no such file or directory")
		}
		return true, nil
	}
	nowFunc = func() time.Time { return clock }
	t.Cleanup(func() {
		execAgentCommand, nowFunc = origExec, origNow
		resetPodCaches()
		resetInjectionRetries()
	})
	return rec, &clock
}

func pythonPod(uid, containerID string) SimplePodInfo {
	return SimplePodInfo{
		Name:             "app-" + uid,
		Namespace:        "default",
		Uid:              uid,
		ContainsApmAgent: true,
		ContainerInfos: []SimpleContainerInfo{{
			ContainerName:         "app",
			ContainerId:           containerID,
			IsApm:                 true,
			ContainerState:        "Running",
			whatapPythonAgentPath: "/whatap-agent/whatap_python",
		}},
	}
}

// injectEvent runs injection the way processPodsMap does for an informer event: the pod info is rebuilt
// from the API object, so ExecProcessed comes from processedContainers.
func injectEvent(p SimplePodInfo) {
	podsMapLock.Lock()
	defer podsMapLock.Unlock()
	for i := range p.ContainerInfos {
		p.ContainerInfos[i].ExecProcessed = isContainerProcessed(p.ContainerInfos[i].ContainerId)
	}
	executeInjection(&p)
}

func storedContainer(t *testing.T, uid string) SimpleContainerInfo {
	t.Helper()
	podsMapLock.RLock()
	defer podsMapLock.RUnlock()
	return podsMap[uid].ContainerInfos[0]
}

func retryAttempts(containerID string) int {
	if v, ok := injectionRetries.Load(containerID); ok {
		return v.(injectionRetry).attempts
	}
	return 0
}

func TestInjectionRetriesFailedExecAfterBackoff(t *testing.T) {
	rec, clock := stubInjection(t, func(call int) bool { return call == 1 })

	injectEvent(pythonPod("u1", "c1"))
	if len(rec.paths) != 1 || rec.paths[0] != "/whatap-agent/whatap_python" {
		t.Fatalf("first pass execs = %v", rec.paths)
	}
	if isContainerProcessed("c1") || storedContainer(t, "u1").ExecProcessed {
		t.Fatal("a failed exec must not mark the container processed")
	}
	if got := retryAttempts("c1"); got != 1 {
		t.Fatalf("attempts = %d, want 1", got)
	}

	retryPendingInjections()
	if len(rec.paths) != 1 {
		t.Fatalf("retried before the backoff elapsed: %v", rec.paths)
	}

	*clock = clock.Add(injectionRetryBaseDelay)
	retryPendingInjections()
	if len(rec.paths) != 2 {
		t.Fatalf("retry after backoff execs = %d, want 2", len(rec.paths))
	}
	if !isContainerProcessed("c1") || !storedContainer(t, "u1").ExecProcessed {
		t.Fatal("a successful retry must mark the container processed")
	}
	if _, pending := injectionRetries.Load("c1"); pending {
		t.Fatal("retry state must be cleared after success")
	}

	*clock = clock.Add(time.Hour)
	retryPendingInjections()
	injectEvent(pythonPod("u1", "c1"))
	if len(rec.paths) != 2 {
		t.Fatalf("processed container executed again: %v", rec.paths)
	}
}

func TestInjectionEventBeforeRetryTimeIsSkipped(t *testing.T) {
	rec, clock := stubInjection(t, func(call int) bool { return call == 1 })

	injectEvent(pythonPod("u1", "c1"))
	*clock = clock.Add(injectionRetryBaseDelay / 2)
	injectEvent(pythonPod("u1", "c1")) // e.g. a readiness update shortly after start
	if len(rec.paths) != 1 {
		t.Fatalf("event before retry time executed: %v", rec.paths)
	}
	*clock = clock.Add(injectionRetryBaseDelay / 2)
	injectEvent(pythonPod("u1", "c1"))
	if len(rec.paths) != 2 || !isContainerProcessed("c1") {
		t.Fatalf("event after retry time: execs %v processed %v", rec.paths, isContainerProcessed("c1"))
	}
}

func TestInjectionGivesUpAfterMaxAttempts(t *testing.T) {
	rec, clock := stubInjection(t, func(int) bool { return true })

	injectEvent(pythonPod("u1", "c1"))
	for n := 0; n < 3*injectionMaxAttempts; n++ {
		*clock = clock.Add(injectionRetryMaxDelay)
		retryPendingInjections()
	}
	if len(rec.paths) != injectionMaxAttempts {
		t.Fatalf("execs = %d, want %d", len(rec.paths), injectionMaxAttempts)
	}
	if !isContainerProcessed("c1") || !storedContainer(t, "u1").ExecProcessed {
		t.Fatal("container must be marked processed after giving up")
	}
	if _, pending := injectionRetries.Load("c1"); pending {
		t.Fatal("retry state must be cleared after giving up")
	}
}

func TestInjectionCountsOneAttemptPerPass(t *testing.T) {
	rec, _ := stubInjection(t, func(int) bool { return true })

	p := pythonPod("u1", "c1")
	p.Labels = map[string]string{"WHATAP_PYTHON_AGENT_PATH": "/whatap-agent/whatap/agent/linux/amd64/whatap_python"}
	injectEvent(p)
	if len(rec.paths) != 2 {
		t.Fatalf("env and label paths should both be tried in one pass, got %v", rec.paths)
	}
	if got := retryAttempts("c1"); got != 1 {
		t.Fatalf("attempts after one pass = %d, want 1", got)
	}
}

func TestInjectionRetryOnlyTouchesDueContainers(t *testing.T) {
	rec, clock := stubInjection(t, func(int) bool { return true })

	injectEvent(pythonPod("u1", "c1"))
	*clock = clock.Add(injectionRetryBaseDelay / 2)
	injectEvent(pythonPod("u2", "c2"))
	*clock = clock.Add(injectionRetryBaseDelay / 2)
	retryPendingInjections()
	if len(rec.paths) != 3 {
		t.Fatalf("execs = %d, want 3 (c1 twice, c2 once)", len(rec.paths))
	}
	if retryAttempts("c1") != 2 || retryAttempts("c2") != 1 {
		t.Fatalf("attempts c1=%d c2=%d, want 2 and 1", retryAttempts("c1"), retryAttempts("c2"))
	}
}

func TestInjectionRetryDelayBackoff(t *testing.T) {
	want := map[int]time.Duration{1: 10 * time.Second, 2: 20 * time.Second, 3: 40 * time.Second, 5: 160 * time.Second, 6: 5 * time.Minute, 20: 5 * time.Minute}
	for attempts, d := range want {
		if got := injectionRetryDelay(attempts); got != d {
			t.Errorf("injectionRetryDelay(%d) = %v, want %v", attempts, got, d)
		}
	}
}

func TestPodCacheCleanupClearsInjectionRetries(t *testing.T) {
	stubInjection(t, func(int) bool { return true })
	origNodename := nodename
	nodename = "test-node"
	defer func() { nodename = origNodename }()

	injectEvent(pythonPod("uid-1", "c1"))
	injectionRetries.Store("c-gone", injectionRetry{attempts: 1})
	injectionRetries.Store("c-live", injectionRetry{attempts: 1})

	removePodFromCache("uid-1")
	if _, pending := injectionRetries.Load("c1"); pending {
		t.Error("removePodFromCache must clear the retry state of the pod's containers")
	}

	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	if err := store.Add(newTestPod("live-pod", "uid-live", "test-node", "c-live")); err != nil {
		t.Fatalf("store.Add: %v", err)
	}
	sweepPodCaches(store)
	if _, pending := injectionRetries.Load("c-gone"); pending {
		t.Error("sweep must clear retry state of containers that no longer exist")
	}
	if _, pending := injectionRetries.Load("c-live"); !pending {
		t.Error("sweep must keep retry state of live containers")
	}
}
