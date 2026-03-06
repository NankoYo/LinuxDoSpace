package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSPAHandlerServesIndexForRoot 验证根路径会返回嵌入的前端入口页。
func TestSPAHandlerServesIndexForRoot(t *testing.T) {
	handler, err := newSPAHandler()
	if err != nil {
		t.Fatalf("newSPAHandler returned error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "LinuxDoSpace") {
		t.Fatalf("expected embedded frontend html, got %q", recorder.Body.String())
	}
}

// TestSPAHandlerFallsBackForClientRoute 验证前端客户端路由会回退到 index.html。
func TestSPAHandlerFallsBackForClientRoute(t *testing.T) {
	handler, err := newSPAHandler()
	if err != nil {
		t.Fatalf("newSPAHandler returned error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/settings", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "LinuxDoSpace") {
		t.Fatalf("expected SPA fallback html, got %q", recorder.Body.String())
	}
}
