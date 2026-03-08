package dashboard

import (
	"testing"
	"time"

	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

func TestRequestMonitorStoresLightweightLogs(t *testing.T) {
	monitor := NewRequestMonitor(20)
	startedAt := time.Now().Add(-250 * time.Millisecond)

	monitor.Start(StartRecord{
		RequestID:     "req-1",
		Model:         "gpt-5.3-codex",
		ThinkingLevel: "low",
		ServiceTier:   "priority",
		StartedAt:     startedAt,
	})
	monitor.Complete(CompleteRecord{
		RequestID:    "req-1",
		StatusCode:   502,
		ErrorMessage: "upstream timeout",
		ResponseBody: `{"error":{"message":"upstream timeout"}}`,
		UsageDetail: coreusage.Detail{
			TotalTokens:      19,
			CachedTokens:     7,
			CacheWriteTokens: 3,
		},
		CompletedAt: startedAt.Add(250 * time.Millisecond),
	})

	logs := monitor.RequestLogs()
	if len(logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(logs))
	}
	if logs[0].Model != "gpt-5.3-codex" {
		t.Fatalf("model = %q", logs[0].Model)
	}
	if logs[0].ThinkingLevel != "low" {
		t.Fatalf("thinking level = %q, want low", logs[0].ThinkingLevel)
	}
	if logs[0].ServiceTier != "priority" {
		t.Fatalf("service tier = %q, want priority", logs[0].ServiceTier)
	}
	if logs[0].TotalTokens != 19 {
		t.Fatalf("total tokens = %d, want 19", logs[0].TotalTokens)
	}
	if logs[0].CacheReadTokens != 7 {
		t.Fatalf("read tokens = %d, want 7", logs[0].CacheReadTokens)
	}
	if logs[0].CacheWriteTokens != 3 {
		t.Fatalf("write tokens = %d, want 3", logs[0].CacheWriteTokens)
	}
	if logs[0].ResponseBody == "" {
		t.Fatal("expected failed response body to be retained")
	}
}

func TestRequestMonitorPublishesSnapshotAndAppend(t *testing.T) {
	monitor := NewRequestMonitor(20)
	subID, stream := monitor.Subscribe(4)
	defer monitor.Unsubscribe(subID)

	snapshot := monitor.SnapshotEvent()
	if snapshot.Type != "snapshot" {
		t.Fatalf("snapshot type = %q", snapshot.Type)
	}
	if len(snapshot.Requests) != 0 {
		t.Fatalf("snapshot live requests = %d, want 0", len(snapshot.Requests))
	}

	startedAt := time.Now()
	monitor.Start(StartRecord{RequestID: "req-2", StartedAt: startedAt})
	select {
	case event := <-stream:
		if event.Type != "live_upsert" {
			t.Fatalf("event type = %q", event.Type)
		}
		if event.Request == nil || event.Request.RequestID != "req-2" {
			t.Fatalf("unexpected live event payload: %+v", event.Request)
		}
		if event.Request.Model != "--" {
			t.Fatalf("initial live model = %q, want --", event.Request.Model)
		}
	case <-time.After(time.Second):
		t.Fatal("expected live_upsert event")
	}

	monitor.Update(UpdateRecord{
		RequestID:     "req-2",
		Model:         "gpt-5.3-codex",
		ThinkingLevel: "high",
		ServiceTier:   "priority",
	})

	select {
	case event := <-stream:
		if event.Type != "live_upsert" {
			t.Fatalf("event type = %q", event.Type)
		}
		if event.Request == nil {
			t.Fatal("expected request payload")
		}
		if event.Request.Model != "gpt-5.3-codex" {
			t.Fatalf("live model = %q, want %q", event.Request.Model, "gpt-5.3-codex")
		}
		if event.Request.ThinkingLevel != "high" {
			t.Fatalf("thinking level = %q, want %q", event.Request.ThinkingLevel, "high")
		}
		if event.Request.ServiceTier != "priority" {
			t.Fatalf("service tier = %q, want %q", event.Request.ServiceTier, "priority")
		}
		if !event.Request.StartTime.Equal(startedAt) {
			t.Fatalf("start time changed after update")
		}
	case <-time.After(time.Second):
		t.Fatal("expected enriched live_upsert event")
	}

	monitor.Complete(CompleteRecord{
		RequestID:  "req-2",
		StatusCode: 200,
		UsageDetail: coreusage.Detail{
			TotalTokens: 5,
		},
	})

	select {
	case event := <-stream:
		if event.Type != "append" {
			t.Fatalf("event type = %q", event.Type)
		}
		if event.RequestID != "req-2" {
			t.Fatalf("append request id = %q, want req-2", event.RequestID)
		}
		if event.Log == nil || event.Log.TotalTokens != 5 {
			t.Fatalf("unexpected event payload: %+v", event.Log)
		}
		if event.Log.ThinkingLevel != "high" {
			t.Fatalf("thinking level = %q, want high", event.Log.ThinkingLevel)
		}
		if event.Log.ServiceTier != "priority" {
			t.Fatalf("service tier = %q, want priority", event.Log.ServiceTier)
		}
	case <-time.After(time.Second):
		t.Fatal("expected append event")
	}
}
