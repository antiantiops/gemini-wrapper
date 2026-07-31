package gemini_impl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"gemini-wrapper/model"
)

type StreamEvent struct {
	Type     string
	Delta    string
	RawEvent map[string]interface{}
}

const maxStreamRecordSize = 4 * 1024 * 1024

// Stream runs agy in documented NDJSON mode. It bypasses result caching and
// singleflight because a stream belongs to exactly one HTTP consumer.
func (s *GeminiService) Stream(ctx context.Context, question, modelName string, emit func(StreamEvent) error) (*model.GeminiStatus, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, fmt.Errorf("question is required")
	}
	args := []string{"--print", question, "--output-format", "stream-json"}
	if resolved, ok := resolveModelName(modelName); ok {
		args = append(args, "--model", resolved)
	}
	if parseEnvBool("ANTIGRAVITY_SKIP_PERMISSIONS", false) {
		args = append(args, "--dangerously-skip-permissions")
	}
	if parseEnvBool("ANTIGRAVITY_SANDBOX", true) {
		args = append(args, "--sandbox")
	}
	cli := strings.TrimSpace(os.Getenv("ANTIGRAVITY_CLI_COMMAND"))
	if cli == "" {
		cli = "agy"
	}
	cmd := exec.CommandContext(ctx, cli, args...)
	if workDir := strings.TrimSpace(os.Getenv("ANTIGRAVITY_WORKDIR")); workDir != "" {
		if info, err := os.Stat(workDir); err == nil && info.IsDir() {
			cmd.Dir = workDir
		}
	}
	home := strings.TrimSpace(os.Getenv("ANTIGRAVITY_HOME"))
	if home == "" {
		home = "/app"
	}
	config := strings.TrimSpace(os.Getenv("ANTIGRAVITY_CONFIG_DIR"))
	if config == "" {
		config = "/app/.gemini"
	}
	cmd.Env = append(os.Environ(), "HOME="+home, "ANTIGRAVITY_CONFIG_DIR="+config, "XDG_CONFIG_HOME="+home)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var stderrBuf strings.Builder
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); _, _ = io.Copy(&stderrBuf, io.LimitReader(stderr, 64*1024)) }()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), maxStreamRecordSize)
	for scanner.Scan() {
		event, err := parseAgyStreamEvent(scanner.Bytes())
		if err != nil {
			_ = cmd.Wait()
			wg.Wait()
			return nil, err
		}
		if err := emit(event); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			wg.Wait()
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		_ = cmd.Wait()
		wg.Wait()
		return nil, err
	}
	err = cmd.Wait()
	wg.Wait()
	if err != nil {
		if out := strings.TrimSpace(stderrBuf.String()); out != "" {
			return nil, fmt.Errorf("antigravity stream failed: %w: %s", err, out)
		}
		return nil, fmt.Errorf("antigravity stream failed: %w", err)
	}
	return nil, nil
}

func parseAgyStreamEvent(line []byte) (StreamEvent, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(line, &raw); err != nil {
		return StreamEvent{}, fmt.Errorf("invalid antigravity stream event: %w", err)
	}
	eventType, _ := raw["event"].(string)
	event := StreamEvent{Type: eventType, RawEvent: raw}
	if nested, ok := raw["step_update"].(map[string]interface{}); ok {
		if delta, ok := nested["text_delta"].(string); ok {
			event.Delta = delta
		}
	}
	if result, ok := raw["result"].(map[string]interface{}); ok {
		if state, _ := result["status"].(string); strings.EqualFold(state, "ERROR") || strings.EqualFold(state, "FAILED") {
			return event, fmt.Errorf("antigravity returned %s", state)
		}
	}
	return event, nil
}
