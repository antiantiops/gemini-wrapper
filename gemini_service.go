package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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

type GeminiStatus struct {
	HTTPStatus int    `json:"httpStatus"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
}

func NewGeminiService() *GeminiService {
	fmt.Println("Gemini service initialized (using headless mode)")
	return &GeminiService{}
}

// Ask sends a question to Gemini CLI using headless mode and returns the response
func (s *GeminiService) Ask(question string, model string) (string, *GeminiStatus, error) {
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
	outputStr := string(output)
	status := detectUpstreamStatus(outputStr, nil)
	if err != nil {
		// Provide helpful error messages for common issues
		if strings.Contains(outputStr, "ModelNotFoundError") || strings.Contains(outputStr, "not found") {
			return "", status, fmt.Errorf("model not found: the model '%s' doesn't exist or isn't available. Use 'gemini-2.5-flash', 'gemini-2.5-flash-lite', 'gemini-2.5-pro', or omit model for auto-selection", model)
		}

		if strings.Contains(outputStr, "authentication") || strings.Contains(outputStr, "auth") {
			return "", status, fmt.Errorf("authentication error: make sure ~/.gemini is mounted correctly and you're authenticated")
		}

		response, ok := parseGeminiOutput(outputStr)
		if ok {
			status = detectUpstreamStatus(outputStr, &response)
			if response.Error != nil {
				answer := strings.TrimSpace(response.Response)
				if status != nil && status.HTTPStatus == http.StatusTooManyRequests && answer != "" {
					return answer, status, nil
				}
				return "", status, fmt.Errorf("gemini error: %s - %s", response.Error.Type, response.Error.Message)
			}

			answer := strings.TrimSpace(response.Response)
			if answer != "" {
				return answer, status, nil
			}
		}

		return "", status, fmt.Errorf("failed to execute gemini CLI: %v (output: %s)", err, outputStr)
	}

	response, ok := parseGeminiOutput(outputStr)
	if !ok {
		// No valid JSON found, return raw output
		fmt.Printf("Warning: No valid JSON found in output\n")
		return strings.TrimSpace(outputStr), status, nil
	}

	status = detectUpstreamStatus(outputStr, &response)

	// Check for errors in response
	if response.Error != nil {
		answer := strings.TrimSpace(response.Response)
		if status != nil && status.HTTPStatus == http.StatusTooManyRequests && answer != "" {
			return answer, status, nil
		}
		errorMsg := fmt.Sprintf("gemini error: %s - %s", response.Error.Type, response.Error.Message)

		// Provide helpful message for common errors
		if strings.Contains(errorMsg, "ModelNotFoundError") || strings.Contains(errorMsg, "not found") {
			return "", status, fmt.Errorf("model not found: the specified model doesn't exist or isn't available. Try using 'gemini-2.5-flash' or don't specify a model for auto-selection")
		}

		return "", status, fmt.Errorf("%s", errorMsg)
	}

	// Return the response text
	answer := strings.TrimSpace(response.Response)
	if answer == "" {
		return "", status, fmt.Errorf("received empty response from gemini")
	}

	fmt.Printf("✓ Response received (%d chars)\n", len(answer))
	return answer, status, nil
}

// AskWithEnv sends a question with custom environment variables
func (s *GeminiService) AskWithEnv(question string, model string, envVars map[string]string) (string, *GeminiStatus, error) {
	// For headless mode, we don't need to modify process env vars
	// Just pass them directly to the command
	return s.Ask(question, model)
}

func parseGeminiOutput(outputStr string) (GeminiResponse, bool) {
	jsonStr, ok := extractLastJSONObject(outputStr)
	if !ok {
		return GeminiResponse{}, false
	}

	var response GeminiResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		fmt.Printf("Warning: Failed to parse JSON response: %v\n", err)
		return GeminiResponse{}, false
	}

	return response, true
}

func detectUpstreamStatus(outputStr string, response *GeminiResponse) *GeminiStatus {
	if response != nil && response.Error != nil && response.Error.Code != 0 {
		status := &GeminiStatus{HTTPStatus: response.Error.Code, Message: response.Error.Message}
		if response.Error.Type != "" {
			status.Code = response.Error.Type
		}
		return status
	}

	if strings.Contains(outputStr, "\"code\": 429") ||
		strings.Contains(outputStr, "status 429") ||
		strings.Contains(outputStr, "Too Many Requests") ||
		strings.Contains(outputStr, "rateLimitExceeded") ||
		strings.Contains(outputStr, "RESOURCE_EXHAUSTED") {
		return &GeminiStatus{
			HTTPStatus: http.StatusTooManyRequests,
			Code:       "RESOURCE_EXHAUSTED",
			Message:    "Upstream rate limited or model capacity exhausted",
		}
	}

	return nil
}

func extractLastJSONObject(outputStr string) (string, bool) {
	depth := 0
	inString := false
	escaped := false
	end := -1

	// Scan backwards to find the last complete JSON object while ignoring braces in strings.
	for i := len(outputStr) - 1; i >= 0; i-- {
		ch := outputStr[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			continue
		}

		if ch == '}' {
			if end == -1 {
				end = i
			}
			depth++
			continue
		}

		if ch == '{' && end != -1 {
			depth--
			if depth == 0 {
				return outputStr[i : end+1], true
			}
		}
	}

	return "", false
}
