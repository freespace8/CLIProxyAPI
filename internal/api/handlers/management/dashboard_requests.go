package management

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/dashboard"
)

func (h *Handler) GetCodexLiveRequests(c *gin.Context) {
	requests := dashboard.DefaultRequestMonitor().LiveRequests()
	c.JSON(http.StatusOK, dashboard.LiveRequestsResponse{
		Requests: requests,
		Count:    len(requests),
	})
}

func (h *Handler) GetCodexRequestLogs(c *gin.Context) {
	logs := dashboard.DefaultRequestMonitor().RequestLogs()
	c.JSON(http.StatusOK, dashboard.RequestLogsResponse{
		Logs:  logs,
		Total: len(logs),
	})
}

func (h *Handler) GetCodexRequestLog(c *gin.Context) {
	rawID := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid log id"})
		return
	}

	logRecord, ok := dashboard.DefaultRequestMonitor().RequestLog(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "request log not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"log": logRecord})
}
