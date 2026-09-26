package middleware_test

import (
	"blog-api/internal/middleware"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCORS_PreflightOptions(t *testing.T) {
	router := gin.New()
	router.Use(middleware.CORS())
	router.POST("/api/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 for OPTIONS preflight, got %d", rec.Code)
	}

	allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin 'http://localhost:3000', got '%s'", allowOrigin)
	}

	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Errorf("expected Access-Control-Allow-Methods header to be set")
	}
}

func TestCORS_StandardRequest(t *testing.T) {
	router := gin.New()
	router.Use(middleware.CORS())
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "https://frontend.example.com")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://frontend.example.com" {
		t.Errorf("expected Access-Control-Allow-Origin 'https://frontend.example.com', got '%s'", allowOrigin)
	}
}

func TestCORS_CustomRequestHeaders(t *testing.T) {
	router := gin.New()
	router.Use(middleware.CORS())
	router.POST("/api/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Headers", "authorization, x-custom-header")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 for OPTIONS preflight, got %d", rec.Code)
	}

	allowHeaders := rec.Header().Get("Access-Control-Allow-Headers")
	if allowHeaders != "authorization, x-custom-header" {
		t.Errorf("expected Access-Control-Allow-Headers 'authorization, x-custom-header', got '%s'", allowHeaders)
	}
}

func TestCORS_NoOrigin(t *testing.T) {
	router := gin.New()
	router.Use(middleware.CORS())
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", allowOrigin)
	}
}

