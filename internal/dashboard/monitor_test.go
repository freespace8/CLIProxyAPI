package dashboard

import "testing"

func TestParseUsageMetricsFallsBackToUpstreamResponse(t *testing.T) {
	totalTokens, readTokens, writeTokens := parseUsageMetrics(
		`{"id":"resp_1","status":"completed"}`,
		"data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":10,\"input_tokens_details\":{\"cached_tokens\":23},\"cache_creation_input_tokens\":11}}}\n\ndata: [DONE]",
	)

	if totalTokens != 10 {
		t.Fatalf("total tokens = %d, want 10", totalTokens)
	}
	if readTokens != 23 {
		t.Fatalf("read tokens = %d, want 23", readTokens)
	}
	if writeTokens != 11 {
		t.Fatalf("write tokens = %d, want 11", writeTokens)
	}
}

func TestParseUsageMetricsReadsSingleJSONPayload(t *testing.T) {
	totalTokens, readTokens, writeTokens := parseUsageMetrics(
		`{"usage":{"total_tokens":19,"prompt_tokens_details":{"cached_tokens":7},"cache_creation_input_tokens":3}}`,
	)

	if totalTokens != 19 {
		t.Fatalf("total tokens = %d, want 19", totalTokens)
	}
	if readTokens != 7 {
		t.Fatalf("read tokens = %d, want 7", readTokens)
	}
	if writeTokens != 3 {
		t.Fatalf("write tokens = %d, want 3", writeTokens)
	}
}
