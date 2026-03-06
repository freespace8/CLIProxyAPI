package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/dashboard"
)

func TestDashboardRequestMonitorMiddlewareTracksResponsesRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := dashboard.NewRequestMonitor(20)
	router := gin.New()
	router.Use(DashboardRequestMonitorMiddleware(store))
	router.POST("/v1/responses", func(c *gin.Context) {
		c.Set("API_REQUEST", []byte("upstream-request"))
		c.Set("API_RESPONSE", []byte("upstream-response"))
		c.JSON(http.StatusOK, gin.H{
			"usage": gin.H{
				"total_tokens": 67,
				"input_tokens_details": gin.H{
					"cached_tokens": 23,
				},
				"cache_creation_input_tokens": 11,
			},
			"ok": true,
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.3-codex","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	live := store.LiveRequests()
	if len(live) != 0 {
		t.Fatalf("live requests = %d, want 0", len(live))
	}

	logs := store.RequestLogs()
	if len(logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(logs))
	}
	if logs[0].Model != "gpt-5.3-codex" {
		t.Fatalf("model = %q", logs[0].Model)
	}
	if !logs[0].IsStreaming {
		t.Fatal("expected streaming request")
	}
	if got := logs[0].RequestHeaders["Authorization"]; got == "Bearer secret-token" || !strings.HasPrefix(got, "Bearer ") {
		t.Fatalf("authorization header not masked: %q", got)
	}
	if logs[0].UpstreamRequest != "upstream-request" || logs[0].UpstreamResponse != "upstream-response" {
		t.Fatalf("unexpected upstream data: %+v", logs[0])
	}
	if logs[0].CacheReadTokens != 23 {
		t.Fatalf("cache read tokens = %d, want 23", logs[0].CacheReadTokens)
	}
	if logs[0].TotalTokens != 67 {
		t.Fatalf("total tokens = %d, want 67", logs[0].TotalTokens)
	}
	if logs[0].CacheWriteTokens != 11 {
		t.Fatalf("cache write tokens = %d, want 11", logs[0].CacheWriteTokens)
	}
}

func TestDashboardRequestMonitorMiddlewareSkipsOtherRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := dashboard.NewRequestMonitor(20)
	router := gin.New()
	router.Use(DashboardRequestMonitorMiddleware(store))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := len(store.RequestLogs()); got != 0 {
		t.Fatalf("logs = %d, want 0", got)
	}
}
