package agy_session

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var ErrNotFound = errors.New("session not found")

type Event map[string]any

type CommandFactory func() (*exec.Cmd, error)

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
	newCmd   CommandFactory
}

type session struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	agyID   string
}

func NewManager() *Manager { return NewManagerWithFactory(defaultCommand) }

func NewManagerWithFactory(factory CommandFactory) *Manager {
	return &Manager{sessions: map[string]*session{}, newCmd: factory}
}

func defaultCommand() (*exec.Cmd, error) {
	binary := strings.TrimSpace(os.Getenv("ANTIGRAVITY_CLI_COMMAND"))
	if binary == "" {
		binary = "agy"
	}
	args := []string{"--input-format", "stream-json", "--output-format", "stream-json", "--print="}
	if os.Getenv("ANTIGRAVITY_SANDBOX") != "false" {
		args = append(args, "--sandbox")
	}
	cmd := exec.Command(binary, args...)
	home := envOr("ANTIGRAVITY_HOME", "/app")
	cmd.Env = append(os.Environ(), "HOME="+home, "ANTIGRAVITY_CONFIG_DIR="+envOr("ANTIGRAVITY_CONFIG_DIR", "/app/.gemini"), "XDG_CONFIG_HOME="+home)
	return cmd, nil
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func (m *Manager) Start() (string, Event, error) {
	cmd, err := m.newCmd()
	if err != nil {
		return "", nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", nil, err
	}
	if err := cmd.Start(); err != nil {
		return "", nil, err
	}
	s := &session{cmd: cmd, stdin: stdin, scanner: newScanner(stdout)}
	init, err := s.next()
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return "", nil, fmt.Errorf("read agy init event: %w", err)
	}
	if init["event"] != "init" {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return "", nil, fmt.Errorf("expected agy init event, got %v", init["event"])
	}
	if conversation, ok := init["conversation_id"].(string); ok {
		s.agyID = conversation
	}
	id, err := newID()
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return "", nil, err
	}
	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	return id, init, nil
}

func (m *Manager) Turn(id, content string, emit func(Event) error) error {
	m.mu.Lock()
	s := m.sessions[id]
	m.mu.Unlock()
	if s == nil {
		return ErrNotFound
	}
	return s.turn(content, emit)
}

func (m *Manager) Exists(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.sessions[id]
	return ok
}

func (m *Manager) Close(id string) error {
	m.mu.Lock()
	s := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if s == nil {
		return ErrNotFound
	}
	_ = s.stdin.Close()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	if err := s.cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return err
		}
	}
	return nil
}

func (s *session) turn(content string, emit func(Event) error) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return errors.New("content is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	request := map[string]any{"event": "user", "message": map[string]string{"role": "user", "content": content}}
	encoded, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if _, err = s.stdin.Write(append(encoded, '\n')); err != nil {
		return err
	}
	for {
		event, err := s.next()
		if err != nil {
			return err
		}
		if err := emit(event); err != nil {
			return err
		}
		if event["event"] == "result" {
			return nil
		}
	}
}

func (s *session) next() (Event, error) {
	if !s.scanner.Scan() {
		if err := s.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	var event Event
	if err := json.Unmarshal(s.scanner.Bytes(), &event); err != nil {
		return nil, fmt.Errorf("decode agy event: %w", err)
	}
	return event, nil
}

func newScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	return scanner
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
