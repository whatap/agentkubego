package pods

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// RunPodInformer가 띄우는 informer가 자기 노드 파드만 list/watch하고,
// 캐시에 넣기 전에 쓰지 않는 메타데이터를 비우는지 API 요청 단위로 확인한다.
// 옵션 함수만 따로 검사하면 factory 생성에서 옵션을 빠뜨려도 잡지 못한다.
func TestNewNodePodInformerScopesToNodeAndStripsMetadata(t *testing.T) {
	pod := newTestPod("p", "uid-1", "node-a", "c1")
	pod.Annotations = map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "{}"}
	pod.ManagedFields = []metav1.ManagedFieldsEntry{{Manager: "kubectl"}}

	var mu sync.Mutex
	var listSelectors, watchSelectors []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/pods" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		if q.Get("watch") == "true" {
			mu.Lock()
			watchSelectors = append(watchSelectors, q.Get("fieldSelector"))
			mu.Unlock()
			// 이벤트 없이 watch를 열어 두고 informer가 멈추면 끝낸다
			w.WriteHeader(http.StatusOK)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			<-r.Context().Done()
			return
		}
		mu.Lock()
		listSelectors = append(listSelectors, q.Get("fieldSelector"))
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(&corev1.PodList{
			TypeMeta: metav1.TypeMeta{Kind: "PodList", APIVersion: "v1"},
			ListMeta: metav1.ListMeta{ResourceVersion: "1"},
			Items:    []corev1.Pod{*pod},
		})
	}))
	defer srv.Close()
	defer srv.CloseClientConnections()

	client, err := kubernetes.NewForConfig(&rest.Config{Host: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	informer := newNodePodInformer(client, "node-a")
	stop := make(chan struct{})
	defer close(stop)
	go informer.Run(stop)
	if !cache.WaitForCacheSync(stop, informer.HasSynced) {
		t.Fatal("informer cache did not sync")
	}
	// watch는 초기 list 직후에 열리므로 잠시 기다린다
	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		watched := len(watchSelectors) > 0
		mu.Unlock()
		if watched || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	lists := append([]string(nil), listSelectors...)
	watches := append([]string(nil), watchSelectors...)
	mu.Unlock()
	if len(lists) == 0 || len(watches) == 0 {
		t.Fatalf("pod list/watch not observed: list=%v watch=%v", lists, watches)
	}
	const want = "spec.nodeName=node-a"
	for _, got := range append(lists, watches...) {
		if got != want {
			t.Fatalf("pod list/watch fieldSelector = %q, want %q (list=%v watch=%v)", got, want, lists, watches)
		}
	}

	obj, exists, err := informer.GetStore().GetByKey("default/p")
	if err != nil || !exists {
		t.Fatalf("pod not cached: exists=%v err=%v", exists, err)
	}
	cached := obj.(*corev1.Pod)
	if cached.Annotations != nil || cached.ManagedFields != nil {
		t.Fatalf("cached pod keeps unused metadata: annotations=%v managedFields=%v", cached.Annotations, cached.ManagedFields)
	}
	if cached.Spec.NodeName != "node-a" || len(cached.Status.ContainerStatuses) != 1 {
		t.Fatalf("cached pod lost fields the handlers read: %+v", cached)
	}
}
