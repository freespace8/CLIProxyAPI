package responses

import (
	"context"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func parseOpenAIResponsesSSEChunk(t *testing.T, chunk string) (string, gjson.Result) {
	t.Helper()

	lines := strings.Split(chunk, "\n")
	if len(lines) < 2 {
		t.Fatalf("unexpected SSE chunk: %q", chunk)
	}

	event := strings.TrimSpace(strings.TrimPrefix(lines[0], "event:"))
	dataLine := strings.TrimSpace(strings.TrimPrefix(lines[1], "data:"))
	if !gjson.Valid(dataLine) {
		t.Fatalf("invalid SSE data JSON: %q", dataLine)
	}

	return event, gjson.Parse(dataLine)
}

func TestConvertOpenAIChatCompletionsResponseToOpenAIResponsesEmitsCompletedUsage(t *testing.T) {
	chunks := [][]byte{
		[]byte(`data: {"id":"chatcmpl-1","created":1,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"你好"}}]}`),
		[]byte(`data: {"id":"chatcmpl-1","created":1,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"，世界"}}]}`),
		[]byte(`data: {"id":"chatcmpl-1","created":1,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":8,"completion_tokens":4,"total_tokens":12,"prompt_tokens_details":{"cached_tokens":2}}}`),
	}

	requestRawJSON := []byte(`{"model":"gpt-5.4","instructions":"请简短回答"}`)
	var param any
	var out []string
	for _, chunk := range chunks {
		out = append(out, ConvertOpenAIChatCompletionsResponseToOpenAIResponses(context.Background(), "gpt-5.4", requestRawJSON, requestRawJSON, chunk, &param)...)
	}

	var completed gjson.Result
	for _, chunk := range out {
		event, data := parseOpenAIResponsesSSEChunk(t, chunk)
		if event == "response.completed" {
			completed = data
			break
		}
	}

	if !completed.Exists() {
		t.Fatal("missing response.completed event")
	}
	if got := completed.Get("response.output.0.content.0.text").String(); got != "你好，世界" {
		t.Fatalf("completed text = %q, want %q", got, "你好，世界")
	}
	if got := completed.Get("response.usage.input_tokens").Int(); got != 8 {
		t.Fatalf("input_tokens = %d, want %d", got, 8)
	}
	if got := completed.Get("response.usage.output_tokens").Int(); got != 4 {
		t.Fatalf("output_tokens = %d, want %d", got, 4)
	}
	if got := completed.Get("response.usage.total_tokens").Int(); got != 12 {
		t.Fatalf("total_tokens = %d, want %d", got, 12)
	}
	if got := completed.Get("response.usage.input_tokens_details.cached_tokens").Int(); got != 2 {
		t.Fatalf("cached_tokens = %d, want %d", got, 2)
	}
}

func TestConvertOpenAIChatCompletionsResponseToOpenAIResponsesEscapesStreamingText(t *testing.T) {
	chunks := [][]byte{
		[]byte("data: {\"id\":\"chatcmpl-escape\",\"created\":2,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"第一行\\n\\\"quoted\\\"\"}}]}"),
		[]byte(`data: {"id":"chatcmpl-escape","created":2,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`),
	}

	var param any
	var out []string
	for _, chunk := range chunks {
		out = append(out, ConvertOpenAIChatCompletionsResponseToOpenAIResponses(context.Background(), "gpt-5.4", nil, nil, chunk, &param)...)
	}

	var deltaSeen bool
	var completed gjson.Result
	for _, chunk := range out {
		event, data := parseOpenAIResponsesSSEChunk(t, chunk)
		switch event {
		case "response.output_text.delta":
			deltaSeen = true
			if got := data.Get("delta").String(); got != "第一行\n\"quoted\"" {
				t.Fatalf("delta = %q, want %q", got, "第一行\n\"quoted\"")
			}
		case "response.completed":
			completed = data
		}
	}

	if !deltaSeen {
		t.Fatal("missing response.output_text.delta event")
	}
	if got := completed.Get("response.output.0.content.0.text").String(); got != "第一行\n\"quoted\"" {
		t.Fatalf("completed text = %q, want %q", got, "第一行\n\"quoted\"")
	}
}

func TestConvertOpenAIChatCompletionsResponseToOpenAIResponsesEscapesFunctionArguments(t *testing.T) {
	chunks := [][]byte{
		[]byte("data: {\"id\":\"chatcmpl-tool\",\"created\":3,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"id\":\"call-1\",\"function\":{\"name\":\"lookup\",\"arguments\":\"{\\\"q\\\":\\\"line1\\\\n\\\\\\\"quoted\\\\\\\"\\\"}\"}}]}}]}"),
		[]byte(`data: {"id":"chatcmpl-tool","created":3,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`),
	}

	var param any
	var out []string
	for _, chunk := range chunks {
		out = append(out, ConvertOpenAIChatCompletionsResponseToOpenAIResponses(context.Background(), "gpt-5.4", nil, nil, chunk, &param)...)
	}

	var deltaSeen bool
	var doneSeen bool
	var completed gjson.Result
	for _, chunk := range out {
		event, data := parseOpenAIResponsesSSEChunk(t, chunk)
		switch event {
		case "response.function_call_arguments.delta":
			deltaSeen = true
			if got := data.Get("delta").String(); got != "{\"q\":\"line1\\n\\\"quoted\\\"\"}" {
				t.Fatalf("delta = %q, want %q", got, "{\"q\":\"line1\\n\\\"quoted\\\"\"}")
			}
		case "response.function_call_arguments.done":
			doneSeen = true
			if got := data.Get("arguments").String(); got != "{\"q\":\"line1\\n\\\"quoted\\\"\"}" {
				t.Fatalf("arguments = %q, want %q", got, "{\"q\":\"line1\\n\\\"quoted\\\"\"}")
			}
		case "response.completed":
			completed = data
		}
	}

	if !deltaSeen {
		t.Fatal("missing response.function_call_arguments.delta event")
	}
	if !doneSeen {
		t.Fatal("missing response.function_call_arguments.done event")
	}
	if got := completed.Get("response.output.0.arguments").String(); got != "{\"q\":\"line1\\n\\\"quoted\\\"\"}" {
		t.Fatalf("completed arguments = %q, want %q", got, "{\"q\":\"line1\\n\\\"quoted\\\"\"}")
	}
}

func BenchmarkConvertOpenAIChatCompletionsResponseToOpenAIResponses(b *testing.B) {
	chunks := [][]byte{
		[]byte(`data: {"id":"chatcmpl-1","created":1,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"第一段"}}]}`),
		[]byte(`data: {"id":"chatcmpl-1","created":1,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"第二段"}}]}`),
		[]byte(`data: {"id":"chatcmpl-1","created":1,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"id":"call-1","function":{"name":"lookup","arguments":"{\"q\":\"cli\"}"}}]}}]}`),
		[]byte(`data: {"id":"chatcmpl-1","created":1,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":64,"completion_tokens":32,"total_tokens":96,"prompt_tokens_details":{"cached_tokens":12}}}`),
	}
	requestRawJSON := []byte(`{"model":"gpt-5.4","instructions":"请简短回答","tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}]}`)

	b.ReportAllocs()
	for b.Loop() {
		var param any
		total := 0
		for _, chunk := range chunks {
			total += len(ConvertOpenAIChatCompletionsResponseToOpenAIResponses(context.Background(), "gpt-5.4", requestRawJSON, requestRawJSON, chunk, &param))
		}
		if total == 0 {
			b.Fatal("translator returned no events")
		}
	}
}
