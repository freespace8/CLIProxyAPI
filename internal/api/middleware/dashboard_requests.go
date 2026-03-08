package middleware

import (
	"bytes"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/dashboard"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/interfaces"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/logging"
	"github.com/tidwall/gjson"
)

const (
	maxDashboardErrorBodyBytes   = 10 * 1024
	dashboardTruncatedBodySuffix = "\n...[truncated]"
)

type dashboardResponseWriter struct {
	gin.ResponseWriter
	body       bytes.Buffer
	statusCode int
	truncated  bool
}

func (w *dashboardResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *dashboardResponseWriter) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	if shouldCaptureDashboardErrorBody(w.statusCode) {
		w.captureBody(data)
	}
	return w.ResponseWriter.Write(data)
}

func (w *dashboardResponseWriter) WriteString(data string) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	if shouldCaptureDashboardErrorBody(w.statusCode) {
		w.captureBody([]byte(data))
	}
	return w.ResponseWriter.WriteString(data)
}

func (w *dashboardResponseWriter) captureBody(data []byte) {
	if len(data) == 0 || w == nil {
		return
	}
	if w.body.Len() >= maxDashboardErrorBodyBytes {
		w.truncated = true
		return
	}
	remaining := maxDashboardErrorBodyBytes - w.body.Len()
	if remaining <= 0 {
		w.truncated = true
		return
	}
	if len(data) > remaining {
		data = data[:remaining]
		w.truncated = true
	}
	_, _ = w.body.Write(data)
}

func (w *dashboardResponseWriter) capturedResponseBody() string {
	if w == nil {
		return ""
	}
	body := strings.TrimSpace(w.body.String())
	if !w.truncated {
		return body
	}
	if body == "" {
		return strings.TrimSpace(dashboardTruncatedBodySuffix)
	}
	return body + dashboardTruncatedBodySuffix
}

func DashboardRequestMonitorMiddleware(store *dashboard.RequestMonitor) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil || !shouldTrackDashboardRequest(c.Request) {
			c.Next()
			return
		}

		requestID := strings.TrimSpace(logging.GetGinRequestID(c))
		if requestID == "" {
			requestID = logging.GenerateRequestID()
			logging.SetGinRequestID(c, requestID)
		}
		startedAt := time.Now()

		writer := &dashboardResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		dashboard.BindRequestMonitor(c, store)

		store.Start(dashboard.StartRecord{
			RequestID:     requestID,
			Model:         dashboard.RequestModel(c),
			ThinkingLevel: dashboard.RequestThinkingLevel(c),
			ServiceTier:   dashboard.RequestServiceTier(c),
			StartedAt:     startedAt,
		})
		c.Next()

		statusCode := writer.statusCode
		if statusCode == 0 {
			statusCode = c.Writer.Status()
		}
		if statusCode > 0 && statusCode < http.StatusBadRequest {
			logging.SkipGinRequestLogging(c)
		}
		store.Complete(dashboard.CompleteRecord{
			RequestID:     requestID,
			Model:         dashboard.RequestModel(c),
			ThinkingLevel: dashboard.RequestThinkingLevel(c),
			ServiceTier:   dashboard.RequestServiceTier(c),
			StatusCode:    statusCode,
			ResponseBody:  writer.capturedResponseBody(),
			ErrorMessage:  resolveDashboardError(statusCode, writer.body.Bytes(), contextErrors(c)),
			UsageDetail:   dashboard.UsageDetail(c),
			CompletedAt:   time.Now(),
		})
	}
}

func shouldTrackDashboardRequest(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	if req.Method != http.MethodPost {
		return false
	}
	path := strings.TrimSpace(req.URL.Path)
	return path == "/v1/responses" || path == "/v1/responses/compact"
}

func contextErrors(c *gin.Context) []*interfaces.ErrorMessage {
	if c == nil {
		return nil
	}
	raw, exists := c.Get("API_RESPONSE_ERROR")
	if !exists {
		return nil
	}
	value, ok := raw.([]*interfaces.ErrorMessage)
	if !ok {
		return nil
	}
	return value
}

func resolveDashboardError(statusCode int, responseBody []byte, apiErrors []*interfaces.ErrorMessage) string {
	for idx := range apiErrors {
		if apiErrors[idx] != nil && apiErrors[idx].Error != nil {
			return strings.TrimSpace(apiErrors[idx].Error.Error())
		}
	}
	if statusCode < http.StatusBadRequest || len(responseBody) == 0 {
		return ""
	}
	if message := strings.TrimSpace(gjson.GetBytes(responseBody, "error.message").String()); message != "" {
		return message
	}
	if message := strings.TrimSpace(gjson.GetBytes(responseBody, "message").String()); message != "" {
		return message
	}
	return strings.TrimSpace(string(responseBody))
}

func shouldCaptureDashboardErrorBody(statusCode int) bool {
	return statusCode >= http.StatusBadRequest
}
