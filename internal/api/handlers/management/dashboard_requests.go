package management

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/dashboard"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/logging"
)

var monitorHeartbeatLine = []byte("{\"type\":\"heartbeat\"}\n")

func (h *Handler) StreamCodexRequestLogs(c *gin.Context) {
	logging.SkipGinRequestLogging(c)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	monitor := dashboard.DefaultRequestMonitor()
	subID, stream := monitor.Subscribe(16)
	defer monitor.Unsubscribe(subID)

	if !writeMonitorStreamEvent(c, monitor.SnapshotEvent(), flusher) {
		return
	}

	heartbeat := time.NewTicker(20 * time.Second)
	liveHeartbeat := time.NewTicker(time.Second)
	defer heartbeat.Stop()
	defer liveHeartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-stream:
			if !ok {
				return
			}
			if !writeMonitorStreamEvent(c, event, flusher) {
				return
			}
		case <-heartbeat.C:
			if monitor.LiveCount() > 0 {
				continue
			}
			if !writeMonitorHeartbeat(c, flusher) {
				return
			}
		case <-liveHeartbeat.C:
			if monitor.LiveCount() == 0 {
				continue
			}
			if !writeMonitorHeartbeat(c, flusher) {
				return
			}
		}
	}
}

func writeMonitorHeartbeat(c *gin.Context, flusher http.Flusher) bool {
	if c == nil {
		return false
	}
	if _, err := c.Writer.Write(monitorHeartbeatLine); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

func writeMonitorStreamEvent(c *gin.Context, event dashboard.MonitorStreamEvent, flusher http.Flusher) bool {
	if c == nil {
		return false
	}
	data, err := json.Marshal(event)
	if err != nil {
		return false
	}
	if _, err = c.Writer.Write(append(data, '\n')); err != nil {
		return false
	}
	flusher.Flush()
	return true
}
