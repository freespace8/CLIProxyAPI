package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/dashboard"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/interfaces"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/logging"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	"github.com/tidwall/gjson"
)

type dashboardResponseWriter struct {
	gin.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

func (w *dashboardResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *dashboardResponseWriter) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	_, _ = w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *dashboardResponseWriter) WriteString(data string) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	w.body.WriteString(data)
	return w.ResponseWriter.WriteString(data)
}

func DashboardRequestMonitorMiddleware(store *dashboard.RequestMonitor) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil || !shouldTrackDashboardRequest(c.Request) {
			c.Next()
			return
		}

		requestInfo, ok := captureDashboardRequest(c)
		if !ok {
			c.Next()
			return
		}

		writer := &dashboardResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		store.Start(requestInfo)
		c.Next()

		completedAt := time.Now()
		statusCode := writer.statusCode
		if statusCode == 0 {
			statusCode = c.Writer.Status()
		}
		store.Complete(dashboard.CompleteRecord{
			RequestID:        requestInfo.RequestID,
			StatusCode:       statusCode,
			ResponseBody:     writer.body.String(),
			UpstreamRequest:  string(contextBytes(c, "API_REQUEST")),
			UpstreamResponse: string(contextBytes(c, "API_RESPONSE")),
			ErrorMessage:     resolveDashboardError(statusCode, writer.body.Bytes(), contextErrors(c)),
			CompletedAt:      completedAt,
		})
	}
}

func captureDashboardRequest(c *gin.Context) (dashboard.StartRecord, bool) {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return dashboard.StartRecord{}, false
	}
	requestID := strings.TrimSpace(logging.GetGinRequestID(c))
	if requestID == "" {
		requestID = logging.GenerateRequestID()
		logging.SetGinRequestID(c, requestID)
	}

	body, ok := readDashboardBody(c.Request)
	if !ok {
		return dashboard.StartRecord{}, false
	}
	return dashboard.StartRecord{
		RequestID:      requestID,
		RequestMethod:  c.Request.Method,
		RequestURL:     maskedRequestURL(c),
		RequestHeaders: maskedRequestHeaders(c.Request.Header),
		RequestBody:    string(body),
		StartedAt:      time.Now(),
	}, true
}

func readDashboardBody(req *http.Request) ([]byte, bool) {
	if req == nil || req.Body == nil {
		return nil, true
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, false
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, true
}

func maskedRequestURL(c *gin.Context) string {
	maskedQuery := util.MaskSensitiveQuery(c.Request.URL.RawQuery)
	if maskedQuery == "" {
		return c.Request.URL.Path
	}
	return c.Request.URL.Path + "?" + maskedQuery
}

func maskedRequestHeaders(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(headers))
	for key, values := range headers {
		result[key] = util.MaskSensitiveHeaderValue(key, strings.Join(values, ", "))
	}
	return result
}

func shouldTrackDashboardRequest(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	path := strings.TrimSpace(req.URL.Path)
	if path == "/v1/responses" || path == "/v1/responses/compact" {
		return true
	}
	if !strings.HasPrefix(path, "/api/provider/") {
		return false
	}
	return strings.HasSuffix(path, "/responses") || strings.HasSuffix(path, "/v1/responses")
}

func contextBytes(c *gin.Context, key string) []byte {
	if c == nil {
		return nil
	}
	raw, exists := c.Get(key)
	if !exists {
		return nil
	}
	value, ok := raw.([]byte)
	if !ok || len(value) == 0 {
		return nil
	}
	return bytes.Clone(value)
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
