package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestProbe_InvalidFromReturns400 验证 /api/cron/next 在 from 字段非法时返回 400。
// 传入 "not-a-time" 作为 from，期望 HTTP 400 而非 422 或 200。
func TestProbe_InvalidFromReturns400(t *testing.T) {
	api := New()
	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	body := []byte(`{"expr":"0 0 1 * *","from":"not-a-time"}`)
	resp, err := http.Post(srv.URL+"/api/cron/next", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("期望 status=400, 实际 status=%d", resp.StatusCode)
	}
}
