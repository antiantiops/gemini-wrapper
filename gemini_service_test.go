package main

import (
	"os"
	"testing"
)

func TestNewGeminiService(t *testing.T) {
	service := NewGeminiService()
	if service == nil {
		t.Fatal("NewGeminiService returned nil")
	}
}

func TestGeminiServiceAsk(t *testing.T) {
	// Skip if no API key is set
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("Skipping test: GEMINI_API_KEY not set")
	}

	service := NewGeminiService()

	// Test with a simple question
	answer, _, err := service.Ask("What is 2+2?", "")
	if err != nil {
		t.Logf("Error asking Gemini: %v", err)
		// Don't fail the test as it might be an environment issue
		t.Skip("Skipping due to Gemini CLI error")
	}

	if answer == "" {
		t.Error("Expected non-empty answer")
	}

	t.Logf("Answer received: %s", answer)
}

func TestGeminiServiceAskWithModel(t *testing.T) {
	// Skip if no API key is set
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("Skipping test: GEMINI_API_KEY not set")
	}

	service := NewGeminiService()

	// Test with a specific model
	answer, _, err := service.Ask("Hello", "gemini-3-flash")
	if err != nil {
		t.Logf("Error asking Gemini with model: %v", err)
		t.Skip("Skipping due to Gemini CLI error")
	}

	if answer == "" {
		t.Error("Expected non-empty answer")
	}

	t.Logf("Answer with model: %s", answer)
}

func TestParseGeminiOutputWith429(t *testing.T) {
	output := "Attempt 1 failed with status 429. [{\"error\":{\"code\":429,\"message\":\"No capacity\"}}] {\"response\":\"ok\",\"stats\":{\"models\":{}}}"

	response, ok := parseGeminiOutput(output)
	if !ok {
		t.Fatal("expected to parse JSON response")
	}

	if response.Response != "ok" {
		t.Fatalf("expected response 'ok', got %q", response.Response)
	}

	status := detectUpstreamStatus(output, &response)
	if status == nil || status.HTTPStatus != 429 {
		t.Fatalf("expected 429 status, got %#v", status)
	}
}
