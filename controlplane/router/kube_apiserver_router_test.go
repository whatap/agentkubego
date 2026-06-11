package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// KAZAA-438 회귀 테스트
// 스크랩 실패/메트릭 부재로 캐시가 비어 있을 때, 컨트롤플레인 객체형 응답의
// buckets/count/sum 이 JSON null 이 아닌 빈 배열([])로 직렬화되어야 한다.
// nil 슬라이스가 null 로 직렬화되면 마스터 에이전트(agentkubejava)에서 NPE 가 발생해
// 해당 수집 주기 전체가 중단된다.

func TestGetApiserverRequestDurationSeconds_EmptyCache_NoNull(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/apiserver-request-duration-seconds", nil)
	rec := httptest.NewRecorder()

	GetApiserverRequestDurationSeconds(rec, req)

	body := strings.TrimSpace(rec.Body.String())
	if strings.Contains(body, "null") {
		t.Fatalf("응답에 null 포함(마스터 NPE 유발): %s", body)
	}

	var resp ApiserverRequestDurationSeconds
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("JSON 파싱 실패: %v, body=%s", err, body)
	}
	if resp.Buckets == nil || resp.Count == nil || resp.Sum == nil {
		t.Fatalf("buckets/count/sum 중 nil 존재: %s", body)
	}
}

func TestGetEtcdRequestDurationSeconds_EmptyCache_NoNull(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/etcd-request-duration-seconds", nil)
	rec := httptest.NewRecorder()

	GetEtcdRequestDurationSeconds(rec, req)

	body := strings.TrimSpace(rec.Body.String())
	if strings.Contains(body, "null") {
		t.Fatalf("응답에 null 포함(마스터 NPE 유발): %s", body)
	}

	var resp EtcdRequestDurationSeconds
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("JSON 파싱 실패: %v, body=%s", err, body)
	}
	if resp.Buckets == nil || resp.Count == nil || resp.Sum == nil {
		t.Fatalf("buckets/count/sum 중 nil 존재: %s", body)
	}
}
