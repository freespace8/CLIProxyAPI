package management

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type flushRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (r *flushRecorder) Flush() {
	r.flushed = true
}

func TestWriteMonitorHeartbeat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	context, _ := gin.CreateTestContext(recorder)

	if !writeMonitorHeartbeat(context, recorder) {
		t.Fatal("writeMonitorHeartbeat returned false")
	}
	if got := recorder.Body.String(); got != string(monitorHeartbeatLine) {
		t.Fatalf("heartbeat body = %q, want %q", got, string(monitorHeartbeatLine))
	}
	if !recorder.flushed {
		t.Fatal("expected flusher to be called")
	}
}
