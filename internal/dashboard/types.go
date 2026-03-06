package dashboard

import "time"

const DefaultLogCapacity = 20

type LiveRequest struct {
	RequestID     string    `json:"requestId"`
	RequestMethod string    `json:"requestMethod"`
	RequestURL    string    `json:"requestUrl"`
	Model         string    `json:"model"`
	Reasoning     string    `json:"reasoning"`
	StartTime     time.Time `json:"startTime"`
	IsStreaming   bool      `json:"isStreaming"`
}

type RequestLogRecord struct {
	ID               int64             `json:"id"`
	RequestID        string            `json:"requestId"`
	RequestMethod    string            `json:"requestMethod"`
	RequestURL       string            `json:"requestUrl"`
	RequestHeaders   map[string]string `json:"requestHeaders"`
	RequestBody      string            `json:"requestBody"`
	ResponseBody     string            `json:"responseBody"`
	UpstreamRequest  string            `json:"upstreamRequest"`
	UpstreamResponse string            `json:"upstreamResponse"`
	Timestamp        time.Time         `json:"timestamp"`
	DurationMs       int64             `json:"durationMs"`
	TotalTokens      int64             `json:"totalTokens"`
	CacheReadTokens  int64             `json:"cacheReadTokens"`
	CacheWriteTokens int64             `json:"cacheWriteTokens"`
	StatusCode       int               `json:"statusCode"`
	Success          bool              `json:"success"`
	Model            string            `json:"model"`
	Reasoning        string            `json:"reasoning"`
	ErrorMessage     string            `json:"errorMessage,omitempty"`
	IsStreaming      bool              `json:"isStreaming"`
}

type StartRecord struct {
	RequestID      string
	RequestMethod  string
	RequestURL     string
	RequestHeaders map[string]string
	RequestBody    string
	StartedAt      time.Time
}

type CompleteRecord struct {
	RequestID        string
	StatusCode       int
	ResponseBody     string
	UpstreamRequest  string
	UpstreamResponse string
	ErrorMessage     string
	CompletedAt      time.Time
}

type LiveRequestsResponse struct {
	Requests []LiveRequest `json:"requests"`
	Count    int           `json:"count"`
}

type RequestLogsResponse struct {
	Logs  []RequestLogRecord `json:"logs"`
	Total int                `json:"total"`
}
