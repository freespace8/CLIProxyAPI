package dashboard

import (
	"sort"
	"strings"
	"sync"
	"time"

	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

type requestSnapshot struct {
	model         string
	thinkingLevel string
	started       time.Time
}

type RequestMonitor struct {
	mu               sync.RWMutex
	capacity         int
	nextID           int64
	nextSubscriberID uint64
	live             map[string]requestSnapshot
	logs             []RequestLogRecord
	subscribers      map[uint64]chan MonitorStreamEvent
}

var defaultMonitor = NewRequestMonitor(DefaultLogCapacity)

func DefaultRequestMonitor() *RequestMonitor { return defaultMonitor }

func NewRequestMonitor(capacity int) *RequestMonitor {
	if capacity <= 0 {
		capacity = DefaultLogCapacity
	}
	return &RequestMonitor{
		capacity:    capacity,
		live:        make(map[string]requestSnapshot),
		subscribers: make(map[uint64]chan MonitorStreamEvent),
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

	m.mu.Lock()
	defer m.mu.Unlock()

	liveRequest := LiveRequest{
		RequestID:     record.RequestID,
		Model:         fallbackModel(record.Model),
		ThinkingLevel: normalizeThinkingLevel(record.ThinkingLevel),
		StartTime:     startedAt,
	}
	m.live[record.RequestID] = requestSnapshot{
		model:         liveRequest.Model,
		thinkingLevel: liveRequest.ThinkingLevel,
		started:       liveRequest.StartTime,
	}
	m.publishLocked(MonitorStreamEvent{
		Type:    "live_upsert",
		Request: cloneLiveRequestPtr(&liveRequest),
	})
}

func (m *RequestMonitor) Update(record UpdateRecord) {
	if m == nil || strings.TrimSpace(record.RequestID) == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot, ok := m.live[record.RequestID]
	if !ok {
		return
	}
	if model := strings.TrimSpace(record.Model); model != "" {
		snapshot.model = model
	}
	if thinkingLevel := normalizeThinkingLevel(record.ThinkingLevel); thinkingLevel != "" {
		snapshot.thinkingLevel = thinkingLevel
	}
	m.live[record.RequestID] = snapshot

	m.publishLocked(MonitorStreamEvent{
		Type: "live_upsert",
		Request: cloneLiveRequestPtr(&LiveRequest{
			RequestID:     record.RequestID,
			Model:         fallbackModel(snapshot.model),
			ThinkingLevel: snapshot.thinkingLevel,
			StartTime:     snapshot.started,
		}),
	})
}

func (m *RequestMonitor) LiveCount() int {
	if m == nil {
		return 0
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.live)
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
	model := snapshot.model
	if trimmed := strings.TrimSpace(record.Model); trimmed != "" {
		model = trimmed
	}
	thinkingLevel := snapshot.thinkingLevel
	if trimmed := normalizeThinkingLevel(record.ThinkingLevel); trimmed != "" {
		thinkingLevel = trimmed
	}

	m.nextID++
	logRecord := RequestLogRecord{
		ID:               m.nextID,
		Timestamp:        completedAt,
		DurationMs:       completedAt.Sub(snapshot.started).Milliseconds(),
		TotalTokens:      resolveTotalTokens(record.UsageDetail),
		CacheReadTokens:  record.UsageDetail.CachedTokens,
		CacheWriteTokens: record.UsageDetail.CacheWriteTokens,
		StatusCode:       record.StatusCode,
		Success:          record.StatusCode > 0 && record.StatusCode < 400,
		Model:            fallbackModel(model),
		ThinkingLevel:    thinkingLevel,
		ErrorMessage:     strings.TrimSpace(record.ErrorMessage),
		ResponseBody:     strings.TrimSpace(record.ResponseBody),
	}
	if logRecord.Success {
		logRecord.ResponseBody = ""
	}

	m.logs = append(m.logs, logRecord)
	if len(m.logs) > m.capacity {
		m.logs = m.logs[len(m.logs)-m.capacity:]
	}
	m.publishLocked(MonitorStreamEvent{
		Type:      "append",
		RequestID: record.RequestID,
		Log:       cloneLogRecordPtr(&logRecord),
	})
}

func (m *RequestMonitor) LiveRequests() []LiveRequest {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	requests := make([]LiveRequest, 0, len(m.live))
	for requestID, record := range m.live {
		requests = append(requests, LiveRequest{
			RequestID:     requestID,
			Model:         fallbackModel(record.model),
			ThinkingLevel: record.thinkingLevel,
			StartTime:     record.started,
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

func (m *RequestMonitor) Subscribe(buffer int) (uint64, <-chan MonitorStreamEvent) {
	if m == nil {
		ch := make(chan MonitorStreamEvent)
		close(ch)
		return 0, ch
	}
	if buffer <= 0 {
		buffer = 16
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextSubscriberID++
	id := m.nextSubscriberID
	ch := make(chan MonitorStreamEvent, buffer)
	m.subscribers[id] = ch
	return id, ch
}

func (m *RequestMonitor) Unsubscribe(id uint64) {
	if m == nil || id == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unsubscribeLocked(id)
}

func (m *RequestMonitor) SnapshotEvent() MonitorStreamEvent {
	return MonitorStreamEvent{
		Type:     "snapshot",
		Requests: m.LiveRequests(),
		Logs:     m.RequestLogs(),
	}
}

func (m *RequestMonitor) unsubscribeLocked(id uint64) {
	ch, exists := m.subscribers[id]
	if !exists {
		return
	}
	delete(m.subscribers, id)
	close(ch)
}

func (m *RequestMonitor) publishLocked(event MonitorStreamEvent) {
	for id, ch := range m.subscribers {
		select {
		case ch <- event:
		default:
			m.unsubscribeLocked(id)
		}
	}
}

func cloneLogRecord(record RequestLogRecord) RequestLogRecord {
	return record
}

func cloneLogRecordPtr(record *RequestLogRecord) *RequestLogRecord {
	if record == nil {
		return nil
	}
	cloned := cloneLogRecord(*record)
	return &cloned
}

func cloneLiveRequest(record LiveRequest) LiveRequest {
	return record
}

func cloneLiveRequestPtr(record *LiveRequest) *LiveRequest {
	if record == nil {
		return nil
	}
	cloned := cloneLiveRequest(*record)
	return &cloned
}

func fallbackModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "--"
	}
	return model
}

func normalizeThinkingLevel(level string) string {
	return strings.ToLower(strings.TrimSpace(level))
}

func resolveTotalTokens(detail coreusage.Detail) int64 {
	if detail.TotalTokens > 0 {
		return detail.TotalTokens
	}
	total := detail.InputTokens + detail.OutputTokens + detail.ReasoningTokens
	if total > 0 {
		return total
	}
	return detail.CachedTokens + detail.CacheWriteTokens
}
