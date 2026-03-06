package dashboard

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

type requestSnapshot struct {
	id        string
	method    string
	url       string
	headers   map[string]string
	body      string
	model     string
	reasoning string
	stream    bool
	started   time.Time
}

type RequestMonitor struct {
	mu       sync.RWMutex
	capacity int
	nextID   int64
	live     map[string]requestSnapshot
	logs     []RequestLogRecord
}

var defaultMonitor = NewRequestMonitor(DefaultLogCapacity)

func DefaultRequestMonitor() *RequestMonitor { return defaultMonitor }

func NewRequestMonitor(capacity int) *RequestMonitor {
	if capacity <= 0 {
		capacity = DefaultLogCapacity
	}
	return &RequestMonitor{
		capacity: capacity,
		live:     make(map[string]requestSnapshot),
	}
}

func (m *RequestMonitor) Start(record StartRecord) {
	if m == nil || strings.TrimSpace(record.RequestID) == "" {
		return
	}
	startedAt := record.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	model, reasoning, stream := parseRequestBody(record.RequestBody)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.live[record.RequestID] = requestSnapshot{
		id:        record.RequestID,
		method:    record.RequestMethod,
		url:       record.RequestURL,
		headers:   cloneHeaders(record.RequestHeaders),
		body:      record.RequestBody,
		model:     model,
		reasoning: reasoning,
		stream:    stream,
		started:   startedAt,
	}
}

func (m *RequestMonitor) Complete(record CompleteRecord) {
	if m == nil || strings.TrimSpace(record.RequestID) == "" {
		return
	}
	completedAt := record.CompletedAt
	if completedAt.IsZero() {
		completedAt = time.Now()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot, ok := m.live[record.RequestID]
	if ok {
		delete(m.live, record.RequestID)
	}
	if !ok {
		return
	}

	m.nextID++
	totalTokens, cacheReadTokens, cacheWriteTokens := parseUsageMetrics(record.ResponseBody, record.UpstreamResponse)
	logRecord := RequestLogRecord{
		ID:               m.nextID,
		RequestID:        snapshot.id,
		RequestMethod:    snapshot.method,
		RequestURL:       snapshot.url,
		RequestHeaders:   cloneHeaders(snapshot.headers),
		RequestBody:      snapshot.body,
		ResponseBody:     record.ResponseBody,
		UpstreamRequest:  record.UpstreamRequest,
		UpstreamResponse: record.UpstreamResponse,
		Timestamp:        completedAt,
		DurationMs:       completedAt.Sub(snapshot.started).Milliseconds(),
		TotalTokens:      totalTokens,
		CacheReadTokens:  cacheReadTokens,
		CacheWriteTokens: cacheWriteTokens,
		StatusCode:       record.StatusCode,
		Success:          record.StatusCode > 0 && record.StatusCode < 400,
		Model:            snapshot.model,
		Reasoning:        snapshot.reasoning,
		ErrorMessage:     strings.TrimSpace(record.ErrorMessage),
		IsStreaming:      snapshot.stream,
	}

	m.logs = append(m.logs, logRecord)
	if len(m.logs) > m.capacity {
		m.logs = m.logs[len(m.logs)-m.capacity:]
	}
}

func (m *RequestMonitor) LiveRequests() []LiveRequest {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	requests := make([]LiveRequest, 0, len(m.live))
	for _, record := range m.live {
		requests = append(requests, LiveRequest{
			RequestID:     record.id,
			RequestMethod: record.method,
			RequestURL:    record.url,
			Model:         record.model,
			Reasoning:     record.reasoning,
			StartTime:     record.started,
			IsStreaming:   record.stream,
		})
	}
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].StartTime.After(requests[j].StartTime)
	})
	return requests
}

func (m *RequestMonitor) RequestLogs() []RequestLogRecord {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	logs := make([]RequestLogRecord, len(m.logs))
	for idx := range m.logs {
		logs[idx] = cloneLogRecord(m.logs[len(m.logs)-1-idx])
	}
	return logs
}

func (m *RequestMonitor) RequestLog(id int64) (RequestLogRecord, bool) {
	if m == nil || id <= 0 {
		return RequestLogRecord{}, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	for idx := len(m.logs) - 1; idx >= 0; idx-- {
		if m.logs[idx].ID == id {
			return cloneLogRecord(m.logs[idx]), true
		}
	}
	return RequestLogRecord{}, false
}

func cloneLogRecord(record RequestLogRecord) RequestLogRecord {
	record.RequestHeaders = cloneHeaders(record.RequestHeaders)
	return record
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}

func parseRequestBody(body string) (string, string, bool) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" || !gjson.Valid(trimmed) {
		return "--", "", false
	}
	model := strings.TrimSpace(gjson.Get(trimmed, "model").String())
	if model == "" {
		model = "--"
	}
	reasoning := strings.TrimSpace(gjson.Get(trimmed, "reasoning.effort").String())
	return model, reasoning, gjson.Get(trimmed, "stream").Bool()
}

func parseUsageMetrics(payloads ...string) (int64, int64, int64) {
	for _, payload := range payloads {
		totalTokens, cacheRead, cacheWrite, ok := parseUsageMetricsFromPayload(payload)
		if ok {
			return totalTokens, cacheRead, cacheWrite
		}
	}
	return 0, 0, 0
}

func parseUsageMetricsFromPayload(payload string) (int64, int64, int64, bool) {
	candidates := splitJSONCandidates(payload)
	for idx := len(candidates) - 1; idx >= 0; idx-- {
		usageNode := firstUsageNode(gjson.Parse(candidates[idx]))
		if !usageNode.Exists() {
			continue
		}
		totalTokens := firstInt(
			usageNode,
			"total_tokens",
			"totalTokenCount",
		)
		if totalTokens == 0 {
			totalTokens = sumPositiveInts(
				firstInt(usageNode, "prompt_tokens", "input_tokens", "promptTokenCount"),
				firstInt(usageNode, "completion_tokens", "output_tokens", "candidatesTokenCount"),
				firstInt(usageNode, "reasoning_tokens", "output_tokens_details.reasoning_tokens", "completion_tokens_details.reasoning_tokens", "thoughtsTokenCount"),
			)
		}
		cacheRead := firstInt(
			usageNode,
			"prompt_tokens_details.cached_tokens",
			"input_tokens_details.cached_tokens",
			"cache_read_input_tokens",
			"cachedContentTokenCount",
		)
		cacheWrite := firstInt(
			usageNode,
			"cache_creation_input_tokens",
		)
		if totalTokens > 0 || cacheRead > 0 || cacheWrite > 0 {
			return totalTokens, cacheRead, cacheWrite, true
		}
	}
	return 0, 0, 0, false
}

func splitJSONCandidates(payload string) []string {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return nil
	}
	if gjson.Valid(trimmed) {
		return []string{trimmed}
	}
	lines := strings.Split(trimmed, "\n")
	candidates := make([]string, 0, len(lines))
	for _, line := range lines {
		candidate := normalizeJSONCandidate(line)
		if candidate == "" || !gjson.Valid(candidate) {
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func normalizeJSONCandidate(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "data:") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
	}
	if trimmed == "[DONE]" {
		return ""
	}
	return trimmed
}

func firstUsageNode(root gjson.Result) gjson.Result {
	paths := []string{
		"usage",
		"response.usage",
		"usageMetadata",
		"response.usageMetadata",
		"usage_metadata",
		"response.usage_metadata",
	}
	for _, path := range paths {
		if result := root.Get(path); result.Exists() {
			return result
		}
	}
	return gjson.Result{}
}

func firstInt(node gjson.Result, paths ...string) int64 {
	for _, path := range paths {
		if result := node.Get(path); result.Exists() {
			return result.Int()
		}
	}
	return 0
}

func sumPositiveInts(values ...int64) int64 {
	var total int64
	for _, value := range values {
		if value > 0 {
			total += value
		}
	}
	return total
}
