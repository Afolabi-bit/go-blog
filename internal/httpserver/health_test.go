package httpserver_test

import (
	"blog-api/internal/httpserver"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestHealthCheck_Healthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", httpserver.NewHealthHandler(&mockPinger{err: nil}))

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["ok"] != true {
		t.Errorf("expected ok: true, got %v", body["ok"])
	}
	if body["database"] != "connected" {
		t.Errorf("expected database: 'connected', got %v", body["database"])
	}
}

func TestHealthCheck_DatabaseDown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", httpserver.NewHealthHandler(&mockPinger{err: errors.New("connection refused")}))

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["ok"] != false {
		t.Errorf("expected ok: false, got %v", body["ok"])
	}
	if body["database"] != "disconnected" {
		t.Errorf("expected database: 'disconnected', got %v", body["database"])
	}
}
