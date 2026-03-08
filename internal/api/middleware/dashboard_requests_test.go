package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/dashboard"
	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

func TestDashboardRequestMonitorMiddlewareTracksResponsesRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := dashboard.NewRequestMonitor(20)
	router := gin.New()
	router.Use(DashboardRequestMonitorMiddleware(store))
	router.POST("/v1/responses", func(c *gin.Context) {
		dashboard.SetRequestModel(c, "gpt-5.3-codex")
		dashboard.SetRequestThinkingLevel(c, "medium")
		dashboard.SetRequestServiceTier(c, dashboard.ResolveRequestServiceTier([]byte(`{"service_tier":"priority"}`)))
		dashboard.SetUsageDetail(c, coreusage.Detail{
			TotalTokens:      67,
			CachedTokens:     23,
			CacheWriteTokens: 11,
		})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	logs := store.RequestLogs()
	if len(logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(logs))
	}
	if logs[0].Model != "gpt-5.3-codex" {
		t.Fatalf("model = %q", logs[0].Model)
	}
	if logs[0].ThinkingLevel != "medium" {
		t.Fatalf("thinking level = %q, want medium", logs[0].ThinkingLevel)
	}
	if logs[0].ServiceTier != "priority" {
		t.Fatalf("service tier = %q, want priority", logs[0].ServiceTier)
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
	if logs[0].ResponseBody != "" {
		t.Fatalf("unexpected response body for success: %q", logs[0].ResponseBody)
	}
}

func TestDashboardRequestMonitorMiddlewareRetainsErrorResponseBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := dashboard.NewRequestMonitor(20)
	router := gin.New()
	router.Use(DashboardRequestMonitorMiddleware(store))
	router.POST("/v1/responses", func(c *gin.Context) {
		dashboard.SetRequestModel(c, "gpt-5.3-codex")
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"message": "upstream timeout",
			},
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	logs := store.RequestLogs()
	if len(logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(logs))
	}
	if logs[0].StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", logs[0].StatusCode, http.StatusBadGateway)
	}
	if logs[0].ErrorMessage != "upstream timeout" {
		t.Fatalf("error message = %q, want %q", logs[0].ErrorMessage, "upstream timeout")
	}
	if logs[0].ResponseBody == "" {
		t.Fatal("expected response body for failed request")
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

func TestDashboardRequestMonitorMiddlewareTruncatesLargeErrorResponseBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := dashboard.NewRequestMonitor(20)
	router := gin.New()
	router.Use(DashboardRequestMonitorMiddleware(store))
	router.POST("/v1/responses", func(c *gin.Context) {
		dashboard.SetRequestModel(c, "gpt-5.3-codex")
		c.Data(http.StatusBadGateway, "application/json", []byte(fmt.Sprintf(`{"error":{"message":"boom"},"payload":"%s"}`, strings.Repeat("x", maxDashboardErrorBodyBytes))))
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	logs := store.RequestLogs()
	if len(logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(logs))
	}
	if !strings.Contains(logs[0].ResponseBody, "[truncated]") {
		t.Fatalf("response body should be truncated: %q", logs[0].ResponseBody)
	}
	if len(logs[0].ResponseBody) > maxDashboardErrorBodyBytes+len(dashboardTruncatedBodySuffix) {
		t.Fatalf("response body length = %d, want <= %d", len(logs[0].ResponseBody), maxDashboardErrorBodyBytes+len(dashboardTruncatedBodySuffix))
	}
}

func TestDashboardRequestMonitorMiddlewarePublishesLiveMetadataFromHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := dashboard.NewRequestMonitor(20)
	subID, stream := store.Subscribe(8)
	defer store.Unsubscribe(subID)

	router := gin.New()
	router.Use(DashboardRequestMonitorMiddleware(store))
	router.POST("/v1/responses", func(c *gin.Context) {
		dashboard.SetRequestModel(c, "gpt-5.3-codex(high)")
		dashboard.SetRequestThinkingLevel(c, "high")
		dashboard.SetRequestServiceTier(c, dashboard.ResolveRequestServiceTier([]byte(`{"service_tier":"priority"}`)))
		dashboard.PublishRequestLiveInfo(c)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var updated *dashboard.LiveRequest
	for i := 0; i < 3; i++ {
		select {
		case event := <-stream:
			if event.Type == "live_upsert" && event.Request != nil && event.Request.Model == "gpt-5.3-codex(high)" {
				req := *event.Request
				updated = &req
			}
		default:
		}
	}

	if updated == nil {
		t.Fatal("expected updated live request payload")
	}
	if updated.ThinkingLevel != "high" {
		t.Fatalf("thinking level = %q, want high", updated.ThinkingLevel)
	}
	if updated.ServiceTier != "priority" {
		t.Fatalf("service tier = %q, want priority", updated.ServiceTier)
	}
}
