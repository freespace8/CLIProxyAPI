package handlers

import "testing"

func TestValidateSSEDataJSON_AllowsMultiLineEvents(t *testing.T) {
	chunk := []byte("event: response.created\n" +
		"data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\"}}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"output\":[],\"usage\":{\"input_tokens\":12,\"output_tokens\":34,\"total_tokens\":46}}}\n\n" +
		"data: [DONE]\n")

	if err := validateSSEDataJSON(chunk); err != nil {
		t.Fatalf("validateSSEDataJSON returned error: %v", err)
	}
}

func BenchmarkValidateSSEDataJSON(b *testing.B) {
	chunk := []byte("event: response.output_text.delta\n" +
		"data: {\"type\":\"response.output_text.delta\",\"sequence_number\":17,\"item_id\":\"msg_resp_1_0\",\"output_index\":0,\"content_index\":0,\"delta\":\"你好，世界\",\"logprobs\":[]}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"sequence_number\":18,\"response\":{\"id\":\"resp_1\",\"object\":\"response\",\"created_at\":1741412345,\"status\":\"completed\",\"background\":false,\"error\":null,\"output\":[{\"id\":\"msg_resp_1_0\",\"type\":\"message\",\"status\":\"completed\",\"content\":[{\"type\":\"output_text\",\"annotations\":[],\"logprobs\":[],\"text\":\"你好，世界\"}],\"role\":\"assistant\"}],\"usage\":{\"input_tokens\":128,\"output_tokens\":256,\"total_tokens\":384,\"input_tokens_details\":{\"cached_tokens\":64},\"output_tokens_details\":{\"reasoning_tokens\":32}}}}\n\n" +
		"data: [DONE]\n")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := validateSSEDataJSON(chunk); err != nil {
			b.Fatalf("validateSSEDataJSON returned error: %v", err)
		}
	}
}
