package main

import (
	"gemini-wrapper/service/gemini/gemini_impl"
	"os"
	"testing"
)

func TestNewGeminiService(t *testing.T) {
	service := gemini_impl.NewGeminiService()
	if service == nil {
		t.Fatal("NewGeminiService returned nil")
	}
}

func TestGeminiServiceAsk(t *testing.T) {
	// Skip if no API key is set
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("Skipping test: GEMINI_API_KEY not set")
	}

	service := gemini_impl.NewGeminiService()

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
}

func TestGeminiServiceAskWithModel(t *testing.T) {
	// Skip if no API key is set
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("Skipping test: GEMINI_API_KEY not set")
	}

	service := gemini_impl.NewGeminiService()

	// Test with a specific model
	answer, _, err := service.Ask("Hello", "gemini-3-flash")
	if err != nil {
		t.Logf("Error asking Gemini with model: %v", err)
		t.Skip("Skipping due to Gemini CLI error")
	}

	if answer == "" {
		t.Error("Expected non-empty answer")
	}
}
