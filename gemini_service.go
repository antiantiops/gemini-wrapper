package main

import (
	"bufio"
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
	mu         sync.Mutex
	ptmx       *os.File
	cmd        *exec.Cmd
	ready      bool
	readyCh    chan bool
	questionCh chan questionRequest
	outputCh   chan string // Channel for output lines from PTY
}

type questionRequest struct {
	question   string
	model      string
	responseCh chan questionResponse
}

type questionResponse struct {
	answer string
	err    error
}

func NewGeminiService() *GeminiService {
	s := &GeminiService{
		readyCh:    make(chan bool, 1),
		questionCh: make(chan questionRequest, 10),
		outputCh:   make(chan string, 100),
	}

	// Start the persistent gemini CLI session
	go s.startGeminiSession()

	// Wait for it to be ready (with timeout)
	select {
	case ready := <-s.readyCh:
		if !ready {
			fmt.Println("WARNING: Gemini CLI failed to start")
		}
	case <-time.After(30 * time.Second):
		fmt.Println("WARNING: Gemini CLI startup timeout")
	}

	return s
}

// startGeminiSession starts a persistent gemini CLI session
func (s *GeminiService) startGeminiSession() {
	fmt.Println("Starting persistent Gemini CLI session...")

	// Prepare the command
	cmd := exec.Command("gemini")

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"HOME=/app",
		"GEMINI_CONFIG_DIR=/app/.gemini",
		"XDG_CONFIG_HOME=/app",
		"USER=root",
	)

	// Create a pseudo-terminal
	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Printf("ERROR: Failed to start gemini: %v\n", err)
		s.readyCh <- false
		return
	}

	s.ptmx = ptmx
	s.cmd = cmd

	// Start output reader
	go s.readOutput()

	// Wait for the prompt to appear (indicates ready to receive input)
	fmt.Println("Waiting for Gemini CLI to authenticate and show prompt...")

	promptReady := false
	timeout := time.After(30 * time.Second)

	for !promptReady {
		select {
		case line := <-s.outputCh:
			// Look for the prompt indicator
			if strings.Contains(line, "Type your message") {
				promptReady = true
				fmt.Println("Prompt detected! Gemini CLI is ready.")
			}
		case <-timeout:
			fmt.Println("WARNING: Timeout waiting for prompt, assuming ready anyway")
			promptReady = true
		}
	}

	// Give it a moment and clear any remaining output
	time.Sleep(1 * time.Second)
	cleared := 0
	for len(s.outputCh) > 0 {
		<-s.outputCh
		cleared++
	}
	fmt.Printf("Cleared %d remaining messages\n", cleared)

	s.ready = true
	s.readyCh <- true
	fmt.Println("Gemini CLI session ready to accept questions!")

	// Process incoming questions
	s.processQuestions()
}

// readOutput continuously reads from PTY and sends to output channel
func (s *GeminiService) readOutput() {
	scanner := bufio.NewScanner(s.ptmx)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		cleanLine := stripANSI(line)

		// Send all lines to output channel for processing
		s.outputCh <- cleanLine

		// Debug output
		if cleanLine != "" && !shouldSkipLine(cleanLine) {
			fmt.Printf("GEMINI: %s\n", cleanLine)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("ERROR reading from gemini: %v\n", err)
	}
}

// shouldSkipLine determines if a line should be filtered out
func shouldSkipLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Skip empty lines
	if trimmed == "" {
		return true
	}

	// Skip ASCII art and decorations
	if strings.Contains(line, "░░░") ||
		strings.Contains(line, "███") ||
		strings.Contains(line, "█████") ||
		strings.Contains(line, "╭──") ||
		strings.Contains(line, "│") ||
		strings.Contains(line, "╰──") {
		return true
	}

	// Skip startup messages and tips
	if strings.Contains(line, "GEMINI") ||
		strings.Contains(line, "with Gemini") ||
		strings.Contains(line, "Tips for getting started") ||
		strings.Contains(line, "Ask questions, edit files, or run commands") ||
		strings.Contains(line, "Be specific for the best results") ||
		strings.Contains(line, "Create GEMINI.md files") ||
		strings.Contains(line, "customize your interactions") ||
		strings.Contains(line, "/help for more information") ||
		strings.Contains(line, "directory.") ||
		strings.Contains(line, "Gemini 3 Flash and Pro") ||
		strings.Contains(line, "Enable \"Preview features\"") ||
		strings.Contains(line, "Learn more at") ||
		strings.Contains(line, "Warning you are running") ||
		strings.Contains(line, "This warning can be disabled") {
		return true
	}

	// Skip status line and prompt indicators
	if strings.Contains(line, "no sandbox") ||
		strings.Contains(line, "Auto (Gemini") ||
		strings.Contains(line, "Type your message") ||
		strings.Contains(line, "/model") ||
		strings.HasPrefix(trimmed, "~") ||
		strings.HasPrefix(trimmed, ">") {
		return true
	}

	// Skip numbered list items (1. 2. 3. 4.)
	if len(trimmed) > 0 && trimmed[0] >= '1' && trimmed[0] <= '9' && len(trimmed) > 1 && trimmed[1] == '.' {
		return true
	}

	return false
}

// isPromptLine detects if this line indicates the prompt is ready
func isPromptLine(line string) bool {
	return strings.Contains(line, "Type your message") ||
		(strings.TrimSpace(line) != "" && strings.HasPrefix(strings.TrimSpace(line), "~"))
}

// processQuestions handles incoming question requests
func (s *GeminiService) processQuestions() {
	for req := range s.questionCh {
		answer, err := s.askQuestion(req.question, req.model)
		req.responseCh <- questionResponse{
			answer: answer,
			err:    err,
		}
	}
}

// askQuestion sends a question to the persistent session
func (s *GeminiService) askQuestion(question string, model string) (string, error) {
	if !s.ready {
		return "", fmt.Errorf("gemini session not ready")
	}

	// Clear any pending output from previous question
	for len(s.outputCh) > 0 {
		<-s.outputCh
	}

	// Change model if needed
	if model != "" {
		_, err := io.WriteString(s.ptmx, "/model "+model+"\n")
		if err != nil {
			return "", fmt.Errorf("failed to set model: %v", err)
		}
		time.Sleep(500 * time.Millisecond)

		// Clear model change output
		for len(s.outputCh) > 0 {
			<-s.outputCh
		}
	}

	// Send the question
	fmt.Printf("Sending question: %s\n", question)
	_, err := io.WriteString(s.ptmx, question+"\n")
	if err != nil {
		return "", fmt.Errorf("failed to write question: %v", err)
	}

	// Collect response until we see the prompt again
	var answer strings.Builder
	collecting := false
	lineCount := 0

	timeout := time.After(90 * time.Second)

	for {
		select {
		case line := <-s.outputCh:
			// Check if we're back at the prompt (end of response)
			if isPromptLine(line) {
				if collecting {
					fmt.Println("Prompt detected, response complete")
					goto done
				}
				continue
			}

			// Skip UI elements and banners
			if shouldSkipLine(line) {
				continue
			}

			// Skip the echo of our question
			if strings.Contains(line, question) {
				continue
			}

			// Check for authentication issues
			if strings.Contains(line, "Waiting for auth") {
				return "", fmt.Errorf("authentication required during question processing")
			}

			// Start collecting any non-skipped content
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				if !collecting {
					collecting = true
					fmt.Printf("Started collecting response\n")
				}
				answer.WriteString(line)
				answer.WriteString("\n")
				lineCount++
			}

		case <-timeout:
			if answer.Len() > 0 {
				return strings.TrimSpace(answer.String()), nil
			}
			return "", fmt.Errorf("timeout waiting for gemini response")
		}
	}

done:

	result := strings.TrimSpace(answer.String())
	if result == "" {
		return "", fmt.Errorf("no response from gemini")
	}

	fmt.Printf("Collected response (%d lines)\n", lineCount)
	return result, nil
}

// Ask sends a question to Gemini CLI and returns the response
func (s *GeminiService) Ask(question string, model string) (string, error) {
	if !s.ready {
		return "", fmt.Errorf("gemini session not ready")
	}

	// Create response channel
	respCh := make(chan questionResponse, 1)

	// Send question request
	s.questionCh <- questionRequest{
		question:   question,
		model:      model,
		responseCh: respCh,
	}

	// Wait for response with timeout
	select {
	case resp := <-respCh:
		return resp.answer, resp.err
	case <-time.After(90 * time.Second):
		return "", fmt.Errorf("timeout waiting for response")
	}
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
