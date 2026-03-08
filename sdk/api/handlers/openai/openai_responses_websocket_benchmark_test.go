package openai

import (
	"fmt"
	"strings"
	"testing"
)

func TestMergeJSONArrayRaw_EmptyInputs(t *testing.T) {
	merged, err := mergeJSONArrayRaw("", "")
	if err != nil {
		t.Fatalf("mergeJSONArrayRaw returned error: %v", err)
	}
	if merged != "[]" {
		t.Fatalf("mergeJSONArrayRaw = %q, want %q", merged, "[]")
	}
}

func TestMergeJSONArrayRaw_PreservesOrder(t *testing.T) {
	existing := `[{"id":"msg-1"},{"id":"msg-2"}]`
	appendRaw := `[{"id":"assistant-1"},{"id":"tool-1"}]`

	merged, err := mergeJSONArrayRaw(existing, appendRaw)
	if err != nil {
		t.Fatalf("mergeJSONArrayRaw returned error: %v", err)
	}
	expected := `[{"id":"msg-1"},{"id":"msg-2"},{"id":"assistant-1"},{"id":"tool-1"}]`
	if merged != expected {
		t.Fatalf("mergeJSONArrayRaw = %q, want %q", merged, expected)
	}
}

func TestMergeJSONArrayRaw_InvalidExistingJSON(t *testing.T) {
	_, err := mergeJSONArrayRaw(`[{"id":"msg-1"}`, `[{"id":"assistant-1"}]`)
	if err == nil {
		t.Fatal("mergeJSONArrayRaw() error = nil, want invalid JSON error")
	}
}

func BenchmarkMergeJSONArrayRaw(b *testing.B) {
	existing := buildJSONArrayRaw("history", 96)
	appendRaw := buildJSONArrayRaw("delta", 12)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged, err := mergeJSONArrayRaw(existing, appendRaw)
		if err != nil {
			b.Fatalf("mergeJSONArrayRaw returned error: %v", err)
		}
		if len(merged) == 0 {
			b.Fatal("mergeJSONArrayRaw returned empty result")
		}
	}
}

func BenchmarkWebsocketJSONPayloadsFromChunk(b *testing.B) {
	chunk := []byte("event: response.created\n" +
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"status\":\"in_progress\",\"output\":[]}}\n\n" +
		"event: response.output_text.delta\n" +
		"data: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_resp_1_0\",\"output_index\":0,\"content_index\":0,\"delta\":\"这是一个很长的中文 token 片段，用来模拟真实流式转发场景。\"}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"status\":\"completed\",\"output\":[{\"id\":\"msg_resp_1_0\",\"type\":\"message\"}],\"usage\":{\"input_tokens\":128,\"output_tokens\":256,\"total_tokens\":384}}}\n\n" +
		"data: [DONE]\n")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		payloads := websocketJSONPayloadsFromChunk(chunk)
		if len(payloads) != 3 {
			b.Fatalf("payloads len = %d, want 3", len(payloads))
		}
	}
}

func buildJSONArrayRaw(prefix string, count int) string {
	if count <= 0 {
		return "[]"
	}
	var builder strings.Builder
	builder.Grow(count * 48)
	builder.WriteByte('[')
	for index := 0; index < count; index++ {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(fmt.Sprintf("{\"type\":\"message\",\"id\":\"%s-%d\",\"role\":\"user\",\"content\":[{\"type\":\"input_text\",\"text\":\"payload-%d\"}]}", prefix, index, index))
	}
	builder.WriteByte(']')
	return builder.String()
}
