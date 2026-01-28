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

	// Wait for authentication to complete
	fmt.Println("Waiting for Gemini CLI to authenticate...")
	time.Sleep(5 * time.Second) // Give it time to pass "Waiting for auth..." phase

	s.ready = true
	s.readyCh <- true
	fmt.Println("Gemini CLI session ready!")

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

	return strings.Contains(line, "░░░") ||
		strings.Contains(line, "╭──") ||
		strings.Contains(line, "│") ||
		strings.Contains(line, "╰──") ||
		strings.Contains(line, "with Gemini") ||
		strings.Contains(line, "Tips for getting started") ||
		strings.Contains(line, "directory.") ||
		strings.Contains(line, "Gemini 3 Flash and Pro") ||
		strings.Contains(line, "Enable \"Preview features\"") ||
		strings.Contains(line, "Learn more at") ||
		strings.Contains(line, "no sandbox") ||
		strings.Contains(line, "Auto (Gemini") ||
		strings.Contains(line, "/model") ||
		strings.HasPrefix(trimmed, "~") ||
		strings.HasPrefix(trimmed, ">") ||
		trimmed == ""
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

	// Collect response
	var answer strings.Builder
	collecting := false
	lineCount := 0
	noOutputCount := 0

	timeout := time.After(90 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case line := <-s.outputCh:
			noOutputCount = 0

			// Skip UI elements
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

			// Start collecting after we see actual content
			if !collecting && strings.TrimSpace(line) != "" {
				collecting = true
			}

			if collecting {
				answer.WriteString(line)
				answer.WriteString("\n")
				lineCount++
			}

			// Stop after we get a reasonable amount of output
			// and see an empty line (end of response)
			if collecting && lineCount > 3 && strings.TrimSpace(line) == "" {
				break
			}

		case <-ticker.C:
			noOutputCount++
			// If we've collected something and no output for 2 seconds, we're done
			if collecting && noOutputCount > 20 {
				break
			}

		case <-timeout:
			if answer.Len() > 0 {
				return strings.TrimSpace(answer.String()), nil
			}
			return "", fmt.Errorf("timeout waiting for gemini response")
		}

		// Break if we have enough output and haven't seen new output in a while
		if collecting && lineCount > 5 && noOutputCount > 10 {
			break
		}
	}

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
