package registry

import "testing"

func TestGemini31FlashLitePreviewRegisteredAcrossCatalogs(t *testing.T) {
	testCases := []struct {
		name   string
		models []*ModelInfo
	}{
		{name: "standard", models: GetGeminiModels()},
		{name: "vertex", models: GetGeminiVertexModels()},
		{name: "cli", models: GetGeminiCLIModels()},
		{name: "ai_studio", models: GetAIStudioModels()},
	}

	for _, testCase := range testCases {
		if !hasModelID(testCase.models, "gemini-3.1-flash-lite-preview") {
			t.Fatalf("%s catalog missing gemini-3.1-flash-lite-preview", testCase.name)
		}
	}

	cfg, ok := GetAntigravityModelConfig()["gemini-3.1-flash-lite-preview"]
	if !ok || cfg == nil || cfg.Thinking == nil {
		t.Fatalf("antigravity config missing gemini-3.1-flash-lite-preview")
	}
}

func hasModelID(models []*ModelInfo, modelID string) bool {
	for _, model := range models {
		if model != nil && model.ID == modelID {
			return true
		}
	}
	return false
}
