package dashboard

import (
	"time"

	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

const DefaultLogCapacity = 20

type LiveRequest struct {
	RequestID     string    `json:"requestId"`
	Model         string    `json:"model"`
	ThinkingLevel string    `json:"thinkingLevel,omitempty"`
	ServiceTier   string    `json:"serviceTier,omitempty"`
	StartTime     time.Time `json:"startTime"`
}

type RequestLogRecord struct {
	ID               int64     `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	FirstTokenMs     *int64    `json:"firstTokenMs,omitempty"`
	DurationMs       int64     `json:"durationMs"`
	TotalTokens      int64     `json:"totalTokens"`
	OutputTokens     int64     `json:"outputTokens"`
	CacheReadTokens  int64     `json:"cacheReadTokens"`
	CacheWriteTokens int64     `json:"cacheWriteTokens"`
	StatusCode       int       `json:"statusCode"`
	Success          bool      `json:"success"`
	Model            string    `json:"model"`
	ThinkingLevel    string    `json:"thinkingLevel,omitempty"`
	ServiceTier      string    `json:"serviceTier,omitempty"`
	ErrorMessage     string    `json:"errorMessage,omitempty"`
	ResponseBody     string    `json:"responseBody,omitempty"`
}

type StartRecord struct {
	RequestID     string
	Model         string
	ThinkingLevel string
	ServiceTier   string
	StartedAt     time.Time
}

type UpdateRecord struct {
	RequestID     string
	Model         string
	ThinkingLevel string
	ServiceTier   string
}

type CompleteRecord struct {
	RequestID     string
	Model         string
	ThinkingLevel string
	ServiceTier   string
	StatusCode    int
	ResponseBody  string
	ErrorMessage  string
	UsageDetail   coreusage.Detail
	FirstTokenAt  time.Time
	CompletedAt   time.Time
}

type MonitorStreamEvent struct {
	Type      string             `json:"type"`
	RequestID string             `json:"requestId,omitempty"`
	Request   *LiveRequest       `json:"request,omitempty"`
	Requests  []LiveRequest      `json:"requests,omitempty"`
	Log       *RequestLogRecord  `json:"log,omitempty"`
	Logs      []RequestLogRecord `json:"logs,omitempty"`
}
