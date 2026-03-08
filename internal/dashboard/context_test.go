package dashboard

import "testing"

func TestResolveRequestThinkingLevel(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		model string
		want  string
	}{
		{
			name: "codex reasoning effort",
			body: `{"reasoning":{"effort":"high"}}`,
			want: "high",
		},
		{
			name: "openai reasoning effort fallback",
			body: `{"reasoning_effort":"medium"}`,
			want: "medium",
		},
		{
			name: "claude output_config effort",
			body: `{"output_config":{"effort":"xhigh"}}`,
			want: "xhigh",
		},
		{
			name:  "model suffix no longer inferred",
			body:  `{}`,
			model: "gpt-5(high)",
			want:  "",
		},
		{
			name:  "gemini thinking level no longer inferred",
			model: "gemini-2.5-pro",
			body:  `{"generationConfig":{"thinkingConfig":{"thinkingLevel":"high"}}}`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveRequestThinkingLevel([]byte(tt.body), tt.model); got != tt.want {
				t.Fatalf("ResolveRequestThinkingLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveRequestServiceTier(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "priority is preserved",
			body: `{"service_tier":"priority"}`,
			want: "priority",
		},
		{
			name: "fast is preserved as original content",
			body: `{"service_tier":"fast"}`,
			want: "fast",
		},
		{
			name: "missing service tier",
			body: `{}`,
			want: "",
		},
		{
			name: "default tier is preserved as original content",
			body: `{"service_tier":"default"}`,
			want: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveRequestServiceTier([]byte(tt.body)); got != tt.want {
				t.Fatalf("ResolveRequestServiceTier() = %q, want %q", got, tt.want)
			}
		})
	}
}
