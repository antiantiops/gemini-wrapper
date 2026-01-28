package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// Regular expression to match ANSI escape codes
var ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b\]11;?\x1b\\|\x1b[=>].*?[a-zA-Z]|\x1b\[[\d;]*[mGKHfJhlr]`)

type GeminiService struct {
	mu sync.Mutex
}

func NewGeminiService() *GeminiService {
	return &GeminiService{}
}

// Ask sends a question to Gemini CLI and returns the response
func (s *GeminiService) Ask(question string, model string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Prepare the command
	args := []string{}
	if model != "" {
		args = append(args, "/model", model)
	}

	cmd := exec.Command("gemini", args...)

	// Set environment variables to tell gemini CLI where to find credentials
	cmd.Env = append(os.Environ(),
		"HOME=/app",
		"GEMINI_CONFIG_DIR=/app/.gemini",
		"XDG_CONFIG_HOME=/app",
		"USER=root",
	)

	// Create a pseudo-terminal to handle interactive TUI
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to start gemini: %v", err)
	}
	defer ptmx.Close()

	// Buffer to collect output
	var outputBuffer bytes.Buffer
	var allOutput bytes.Buffer

	// Channel to signal completion
	done := make(chan error)
	answerStarted := false
	collectingAnswer := false
	lineCount := 0

	// Read output in a goroutine
	go func() {
		scanner := bufio.NewScanner(ptmx)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024) // Increase buffer size for large responses

		for scanner.Scan() {
			line := scanner.Text()
			allOutput.WriteString(line + "\n")

			// Strip ANSI escape codes
			cleanLine := stripANSI(line)
			lineCount++

			// Check for authentication issues
			if strings.Contains(cleanLine, "Waiting for auth") {
				done <- fmt.Errorf("authentication required: gemini CLI is not authenticated in container. Make sure ~/.gemini is mounted correctly")
				return
			}

			// Skip the ASCII art, warnings, TUI boxes, and initial prompts
			if strings.Contains(cleanLine, "░░░") || // ASCII art
				strings.Contains(cleanLine, "╭──") || // Box top
				strings.Contains(cleanLine, "│") || // Box sides
				strings.Contains(cleanLine, "╰──") || // Box bottom
				strings.Contains(cleanLine, "GEMINI") ||
				strings.Contains(cleanLine, "with Gemini") ||
				strings.Contains(cleanLine, "Tips for getting started") ||
				strings.Contains(cleanLine, "Ask questions") ||
				strings.Contains(cleanLine, "Be specific") ||
				strings.Contains(cleanLine, "Create GEMINI.md") ||
				strings.Contains(cleanLine, "/help for more information") ||
				strings.Contains(cleanLine, "Warning you are running") ||
				strings.Contains(cleanLine, "This warning can be disabled") ||
				strings.Contains(cleanLine, "directory.") ||
				strings.Contains(cleanLine, "Gemini 3 Flash and Pro") ||
				strings.Contains(cleanLine, "Enable \"Preview features\"") ||
				strings.Contains(cleanLine, "Learn more at") ||
				strings.Contains(cleanLine, "no sandbox") ||
				strings.Contains(cleanLine, "Auto (Gemini") ||
				strings.Contains(cleanLine, "/model") ||
				strings.HasPrefix(strings.TrimSpace(cleanLine), "~") ||
				strings.HasPrefix(strings.TrimSpace(cleanLine), ">") ||
				strings.Contains(cleanLine, "Type your message or @path/to/file") {
				continue
			}

			// Skip empty lines
			if strings.TrimSpace(cleanLine) == "" {
				continue
			}

			// Detect when answer starts (non-empty line after prompt)
			if !answerStarted {
				answerStarted = true
				collectingAnswer = true
			}

			// Collect the answer
			if collectingAnswer {
				outputBuffer.WriteString(cleanLine)
				outputBuffer.WriteString("\n")
			}

			// Stop collecting when we see indicators that response is complete
			// Usually after getting substantial output (more than 100 lines) or seeing the next prompt
			if answerStarted && lineCount > 100 && (strings.Contains(cleanLine, ">") || strings.Contains(cleanLine, "~")) {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			done <- fmt.Errorf("scanner error: %v", err)
			return
		}

		done <- nil
	}()

	// Wait for the prompt to appear (give it a moment to initialize)
	time.Sleep(3 * time.Second)

	// Send the question
	_, err = io.WriteString(ptmx, question+"\n")
	if err != nil {
		return "", fmt.Errorf("failed to write question: %v", err)
	}

	// Wait for response with timeout
	select {
	case err := <-done:
		if err != nil {
			return "", err
		}
	case <-time.After(90 * time.Second):
		// Try to capture what we have so far
		if outputBuffer.Len() > 0 {
			return strings.TrimSpace(outputBuffer.String()), nil
		}
		return "", fmt.Errorf("timeout waiting for gemini response")
	}

	// Send Ctrl+C to exit gracefully
	ptmx.Write([]byte{3})
	time.Sleep(500 * time.Millisecond)

	// Wait for process to finish (with timeout)
	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- cmd.Wait()
	}()

	select {
	case <-cmdDone:
		// Process finished
	case <-time.After(2 * time.Second):
		// Force kill if it doesn't exit
		cmd.Process.Kill()
	}

	answer := strings.TrimSpace(outputBuffer.String())
	if answer == "" {
		// Try to extract something useful from all output
		allText := allOutput.String()
		if allText != "" {
			return "", fmt.Errorf("no clear response from gemini. Raw output:\n%s", allText)
		}
		return "", fmt.Errorf("no response from gemini")
	}

	return answer, nil
}

// AskWithEnv sends a question with custom environment variables
func (s *GeminiService) AskWithEnv(question string, model string, envVars map[string]string) (string, error) {
	// Note: Don't lock here, Ask() will lock
	// Set environment variables temporarily
	for key, value := range envVars {
		os.Setenv(key, value)
	}
	defer func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()

	return s.Ask(question, model)
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(str string) string {
	return ansiEscapeRegex.ReplaceAllString(str, "")
}
