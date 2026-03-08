package handlers

import (
	"strings"
	"testing"
)

func TestBuildErrorResponseBody_PreservesUpstreamJSON(t *testing.T) {
	payload := `{"error":{"message":"upstream","type":"server_error"}}`

	got := BuildErrorResponseBody(502, payload)

	if string(got) != payload {
		t.Fatalf("BuildErrorResponseBody() = %q, want %q", string(got), payload)
	}
}

func TestValidateSSEDataJSON_RejectsInvalidJSON(t *testing.T) {
	chunk := []byte("event: response.completed\n" +
		"data: {\"type\":\"response.completed\"\n\n")

	err := validateSSEDataJSON(chunk)
	if err == nil {
		t.Fatal("validateSSEDataJSON() error = nil, want invalid JSON error")
	}
	if !strings.Contains(err.Error(), "invalid SSE data JSON") {
		t.Fatalf("validateSSEDataJSON() error = %q, want invalid SSE data JSON", err.Error())
	}
}
