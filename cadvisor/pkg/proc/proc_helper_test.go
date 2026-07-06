package proc

import (
	"testing"
	"time"
)

func resetPidLookup() {
	containerPidLookupMutex.Lock()
	defer containerPidLookupMutex.Unlock()
	containerPidLookup = map[string][]int64{}
	lastPidLookupSweepAt = 0
}

func TestRemoveContainerPid(t *testing.T) {
	resetPidLookup()
	now := time.Now().Unix()
	containerPidLookupMutex.Lock()
	containerPidLookup["c1"] = []int64{100, now}
	containerPidLookupMutex.Unlock()

	RemoveContainerPid("c1")
	RemoveContainerPid("not-exist") // 없는 키 제거는 no-op

	containerPidLookupMutex.RLock()
	defer containerPidLookupMutex.RUnlock()
	if _, ok := containerPidLookup["c1"]; ok {
		t.Errorf("c1 should be removed")
	}
}

func TestSweepStaleContainerPids(t *testing.T) {
	resetPidLookup()
	now := time.Now().Unix()
	containerPidLookupMutex.Lock()
	containerPidLookup["fresh"] = []int64{100, now - 10}
	containerPidLookup["stale"] = []int64{200, now - pidCacheMaxAgeSeconds - 1}
	containerPidLookup["broken"] = []int64{300} // 타임스탬프 없는 비정상 항목
	containerPidLookupMutex.Unlock()

	sweepStaleContainerPids(now)

	containerPidLookupMutex.RLock()
	defer containerPidLookupMutex.RUnlock()
	if _, ok := containerPidLookup["fresh"]; !ok {
		t.Errorf("fresh entry should survive sweep")
	}
	if _, ok := containerPidLookup["stale"]; ok {
		t.Errorf("stale entry should be swept")
	}
	if _, ok := containerPidLookup["broken"]; ok {
		t.Errorf("broken entry should be swept")
	}
	if lastPidLookupSweepAt != now {
		t.Errorf("lastPidLookupSweepAt should be updated to %d, got %d", now, lastPidLookupSweepAt)
	}
}

func TestSweepStaleContainerPidsRespectsInterval(t *testing.T) {
	resetPidLookup()
	now := time.Now().Unix()
	containerPidLookupMutex.Lock()
	lastPidLookupSweepAt = now - 1 // 직전에 sweep이 돌았다면
	containerPidLookup["stale"] = []int64{200, now - pidCacheMaxAgeSeconds - 1}
	containerPidLookupMutex.Unlock()

	sweepStaleContainerPids(now)

	containerPidLookupMutex.RLock()
	defer containerPidLookupMutex.RUnlock()
	if _, ok := containerPidLookup["stale"]; !ok {
		t.Errorf("sweep should be skipped within interval")
	}
}
