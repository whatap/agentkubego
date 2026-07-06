package pods

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

func resetPodCaches() {
	podsMapLock.Lock()
	podsMap = make(map[string]SimplePodInfo)
	podsMapLock.Unlock()
	processedContainers.Range(func(key, _ interface{}) bool {
		processedContainers.Delete(key)
		return true
	})
}

func newTestPod(name, uid, node, containerId string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID(uid),
		},
		Spec: corev1.PodSpec{NodeName: node},
	}
	if containerId != "" {
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{Name: "app", ContainerID: "containerd://" + containerId},
		}
	}
	return pod
}

func TestRemovePodFromCache(t *testing.T) {
	resetPodCaches()
	podsMapLock.Lock()
	podsMap["uid-1"] = SimplePodInfo{
		Uid: "uid-1",
		ContainerInfos: []SimpleContainerInfo{
			{ContainerName: "app", ContainerId: "c1"},
			{ContainerName: "no-id", ContainerId: ""},
		},
	}
	podsMapLock.Unlock()
	processedContainers.Store("c1", true)

	removePodFromCache("uid-1")
	removePodFromCache("uid-unknown") // 없는 UID는 no-op

	podsMapLock.RLock()
	_, podExists := podsMap["uid-1"]
	podsMapLock.RUnlock()
	if podExists {
		t.Errorf("uid-1 should be removed from podsMap")
	}
	if isContainerProcessed("c1") {
		t.Errorf("c1 should be removed from processedContainers")
	}
}

func TestPodDeleteHandlerWithTombstone(t *testing.T) {
	resetPodCaches()
	origNodename := nodename
	nodename = "test-node"
	defer func() { nodename = origNodename }()

	podsMapLock.Lock()
	podsMap["uid-1"] = SimplePodInfo{Uid: "uid-1"}
	podsMapLock.Unlock()

	pod := newTestPod("pod-1", "uid-1", "test-node", "c1")
	podDeleteHandler(cache.DeletedFinalStateUnknown{Key: "default/pod-1", Obj: pod})

	podsMapLock.RLock()
	_, podExists := podsMap["uid-1"]
	podsMapLock.RUnlock()
	if podExists {
		t.Errorf("tombstone delete should remove pod from podsMap")
	}
}

func TestPodDeleteHandlerIgnoresOtherNode(t *testing.T) {
	resetPodCaches()
	origNodename := nodename
	nodename = "test-node"
	defer func() { nodename = origNodename }()

	podsMapLock.Lock()
	podsMap["uid-1"] = SimplePodInfo{Uid: "uid-1"}
	podsMapLock.Unlock()

	podDeleteHandler(newTestPod("pod-1", "uid-1", "other-node", "c1"))

	podsMapLock.RLock()
	_, podExists := podsMap["uid-1"]
	podsMapLock.RUnlock()
	if !podExists {
		t.Errorf("delete event for another node's pod should not touch podsMap")
	}
}

func TestSweepPodCaches(t *testing.T) {
	resetPodCaches()
	origNodename := nodename
	nodename = "test-node"
	defer func() { nodename = origNodename }()

	// informer store에는 live 파드만 존재
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	if err := store.Add(newTestPod("live-pod", "uid-live", "test-node", "c-live")); err != nil {
		t.Fatalf("store.Add: %v", err)
	}

	podsMapLock.Lock()
	podsMap["uid-live"] = SimplePodInfo{Uid: "uid-live"}
	podsMap["uid-gone"] = SimplePodInfo{Uid: "uid-gone"} // 삭제 이벤트를 놓친 파드
	podsMapLock.Unlock()
	processedContainers.Store("c-live", true)
	processedContainers.Store("c-gone", true) // 죽은 컨테이너 ID

	sweepPodCaches(store)

	podsMapLock.RLock()
	_, liveExists := podsMap["uid-live"]
	_, goneExists := podsMap["uid-gone"]
	podsMapLock.RUnlock()
	if !liveExists {
		t.Errorf("live pod should survive sweep")
	}
	if goneExists {
		t.Errorf("gone pod should be swept")
	}
	if !isContainerProcessed("c-live") {
		t.Errorf("live container should survive sweep")
	}
	if isContainerProcessed("c-gone") {
		t.Errorf("gone container should be swept")
	}
}
