package dashboard

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/logging"
	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
	"github.com/tidwall/gjson"
)

const (
	requestModelContextKey         = "DASHBOARD_REQUEST_MODEL"
	requestThinkingLevelContextKey = "DASHBOARD_REQUEST_THINKING_LEVEL"
	requestMonitorContextKey       = "DASHBOARD_REQUEST_MONITOR"
	usageDetailContextKey          = "DASHBOARD_USAGE_DETAIL"
)

func SetRequestModel(c *gin.Context, model string) {
	if c == nil {
		return
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return
	}
	c.Set(requestModelContextKey, model)
}

func RequestModel(c *gin.Context) string {
	if c == nil {
		return ""
	}
	raw, exists := c.Get(requestModelContextKey)
	if !exists {
		return ""
	}
	model, _ := raw.(string)
	return strings.TrimSpace(model)
}

func SetRequestThinkingLevel(c *gin.Context, level string) {
	if c == nil {
		return
	}
	level = normalizeThinkingLevel(level)
	if level == "" {
		return
	}
	c.Set(requestThinkingLevelContextKey, level)
}

func RequestThinkingLevel(c *gin.Context) string {
	if c == nil {
		return ""
	}
	raw, exists := c.Get(requestThinkingLevelContextKey)
	if !exists {
		return ""
	}
	level, _ := raw.(string)
	return normalizeThinkingLevel(level)
}

func BindRequestMonitor(c *gin.Context, monitor *RequestMonitor) {
	if c == nil || monitor == nil {
		return
	}
	c.Set(requestMonitorContextKey, monitor)
}

// PublishRequestLiveInfo 只更新实时监控所需的轻量元数据，不做额外 body 读取。
func PublishRequestLiveInfo(c *gin.Context) {
	if c == nil {
		return
	}
	raw, exists := c.Get(requestMonitorContextKey)
	if !exists {
		return
	}
	monitor, _ := raw.(*RequestMonitor)
	if monitor == nil {
		return
	}
	requestID := strings.TrimSpace(logging.GetGinRequestID(c))
	if requestID == "" {
		return
	}
	monitor.Update(UpdateRecord{
		RequestID:     requestID,
		Model:         RequestModel(c),
		ThinkingLevel: RequestThinkingLevel(c),
	})
}

// ResolveRequestThinkingLevel 复用 handler 已拿到的 rawJSON，只读取显式 effort 字段。
// 这里不再根据 suffix / budget / thinkingLevel 猜测，避免实时监控展示与真实请求口径不一致。
func ResolveRequestThinkingLevel(rawJSON []byte, model string) string {
	_ = model
	for _, path := range []string{
		"reasoning.effort",
		"reasoning_effort",
		"output_config.effort",
	} {
		if value := normalizeThinkingLevel(gjson.GetBytes(rawJSON, path).String()); value != "" {
			return value
		}
	}
	return ""
}

func SetUsageDetail(c *gin.Context, detail coreusage.Detail) {
	if c == nil {
		return
	}
	c.Set(usageDetailContextKey, detail)
}

func UsageDetail(c *gin.Context) coreusage.Detail {
	if c == nil {
		return coreusage.Detail{}
	}
	raw, exists := c.Get(usageDetailContextKey)
	if !exists {
		return coreusage.Detail{}
	}
	detail, _ := raw.(coreusage.Detail)
	return detail
}
