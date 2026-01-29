package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type GeminiService struct {
	mu sync.Mutex
}

// GeminiResponse represents the JSON response from gemini CLI headless mode
type GeminiResponse struct {
	Response string `json:"response"`
	Stats    struct {
		Models map[string]struct {
			Tokens struct {
				Total int `json:"total"`
			} `json:"tokens"`
		} `json:"models"`
	} `json:"stats"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Code    int    `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

func NewGeminiService() *GeminiService {
	fmt.Println("Gemini service initialized (using headless mode)")
	return &GeminiService{}
}

// Ask sends a question to Gemini CLI using headless mode and returns the response
func (s *GeminiService) Ask(question string, model string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Printf("Processing question: %q (model: %s)\n", question, model)

	// Prepare the command arguments
	args := []string{
		"--prompt", question,
		"--output-format", "json",
	}

	// Add model if specified
	if model != "" {
		args = append(args, "--model", model)
	}

	// Create command
	cmd := exec.Command("gemini", args...)

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"HOME=/app",
		"GEMINI_CONFIG_DIR=/app/.gemini",
		"XDG_CONFIG_HOME=/app",
	)

	// Run command and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute gemini CLI: %v (output: %s)", err, string(output))
	}

	// The output may contain debug messages before the JSON
	// Find the JSON object (starts with { and ends with })
	outputStr := string(output)
	jsonStart := strings.Index(outputStr, "{")
	jsonEnd := strings.LastIndex(outputStr, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonStart >= jsonEnd {
		// No valid JSON found, return raw output
		fmt.Printf("Warning: No valid JSON found in output\n")
		return strings.TrimSpace(outputStr), nil
	}

	// Extract just the JSON part
	jsonStr := outputStr[jsonStart : jsonEnd+1]

	// Parse JSON response
	var response GeminiResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		// If JSON parsing fails, return raw output
		fmt.Printf("Warning: Failed to parse JSON response: %v\n", err)
		return strings.TrimSpace(outputStr), nil
	}

	// Check for errors in response
	if response.Error != nil {
		return "", fmt.Errorf("gemini error: %s - %s", response.Error.Type, response.Error.Message)
	}

	// Return the response text
	answer := strings.TrimSpace(response.Response)
	if answer == "" {
		return "", fmt.Errorf("received empty response from gemini")
	}

	fmt.Printf("✓ Response received (%d chars)\n", len(answer))
	return answer, nil
}

// AskWithEnv sends a question with custom environment variables
func (s *GeminiService) AskWithEnv(question string, model string, envVars map[string]string) (string, error) {
	// For headless mode, we don't need to modify process env vars
	// Just pass them directly to the command
	return s.Ask(question, model)
}
