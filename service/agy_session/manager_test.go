package agy_session

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func fakeCommand(events string) CommandFactory {
	return func() (*exec.Cmd, error) {
		cmd := exec.Command("sh", "-c", "printf '%s' \"$EVENTS\"; cat >/dev/null")
		cmd.Env = append(cmd.Environ(), "EVENTS="+events)
		return cmd, nil
	}
}

func TestManagerForwardsNativeToolEvents(t *testing.T) {
	events := `{"event":"init","conversation_id":"agy-1"}
{"event":"step_update","step_update":{"step_type":"tool","state":"ACTIVE","tool_name":"list_dir"}}
{"event":"step_update","step_update":{"step_type":"tool","state":"DONE","tool_name":"list_dir","tool_info":{"output":"a.txt"}}}
{"event":"result","result":{"status":"SUCCESS","response":"done"}}
`
	manager := NewManagerWithFactory(fakeCommand(events))
	id, init, err := manager.Start()
	if err != nil {
		t.Fatal(err)
	}
	if init["conversation_id"] != "agy-1" {
		t.Fatalf("init=%v", init)
	}
	var got []Event
	err = manager.Turn(context.Background(), id, "list files", func(event Event) error {
		got = append(got, event)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("events=%d, want 3", len(got))
	}
	step := got[0]["step_update"].(map[string]any)
	if step["step_type"] != "tool" || step["tool_name"] != "list_dir" {
		t.Fatalf("tool event=%v", step)
	}
	if got[2]["event"] != "result" {
		t.Fatalf("final=%v", got[2])
	}
	_ = manager.Close(id)
}

func TestManagerRejectsUnknownSession(t *testing.T) {
	manager := NewManagerWithFactory(func() (*exec.Cmd, error) { return nil, nil })
	err := manager.Turn(context.Background(), "missing", "hello", func(Event) error { return nil })
	if err != ErrNotFound {
		t.Fatalf("error=%v", err)
	}
	if manager.Exists("missing") {
		t.Fatal("missing session exists")
	}
}

func TestDefaultCommandUsesNativeStreamProtocol(t *testing.T) {
	t.Setenv("ANTIGRAVITY_CLI_COMMAND", "fake-agy")
	t.Setenv("ANTIGRAVITY_SANDBOX", "true")
	cmd, err := defaultCommand()
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(cmd.Args, " ")
	for _, expected := range []string{"--input-format stream-json", "--output-format stream-json", "--print=", "--sandbox"} {
		if !strings.Contains(args, expected) {
			t.Fatalf("args %q missing %q", args, expected)
		}
	}
}

func TestManagerPurgesSessionOnClientDisconnect(t *testing.T) {
	events := `{"event":"init","conversation_id":"agy-disc"}
{"event":"step_update","step_update":{"step_type":"tool","state":"ACTIVE","tool_name":"run_cmd"}}
{"event":"result","result":{"status":"SUCCESS","response":"done"}}
`
	manager := NewManagerWithFactory(fakeCommand(events))
	id, _, err := manager.Start()
	if err != nil {
		t.Fatal(err)
	}

	simulatedErr := errors.New("client disconnected")
	err = manager.Turn(context.Background(), id, "run command", func(event Event) error {
		// Fail on first streamed event to simulate client disconnect
		return simulatedErr
	})
	if !errors.Is(err, simulatedErr) {
		t.Fatalf("expected simulatedErr, got %v", err)
	}

	// Session must be purged to prevent unread buffer leakage
	if manager.Exists(id) {
		t.Fatalf("session %s should have been purged on emit error", id)
	}
}

func TestManagerPurgesSessionOnContextCancel(t *testing.T) {
	events := `{"event":"init","conversation_id":"agy-ctx"}
`
	// Command outputs init, then hangs waiting on stdin
	manager := NewManagerWithFactory(fakeCommand(events))
	id, _, err := manager.Start()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = manager.Turn(ctx, id, "waiting turn", func(event Event) error {
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}

	// Session must be purged
	if manager.Exists(id) {
		t.Fatalf("session %s should have been purged on context cancellation", id)
	}
}

func TestManagerCloseAll(t *testing.T) {
	events := `{"event":"init","conversation_id":"agy-closeall"}
`
	manager := NewManagerWithFactory(fakeCommand(events))
	id1, _, err := manager.Start()
	if err != nil {
		t.Fatal(err)
	}
	id2, _, err := manager.Start()
	if err != nil {
		t.Fatal(err)
	}

	manager.CloseAll()
	if manager.Exists(id1) || manager.Exists(id2) {
		t.Fatal("all sessions should be closed")
	}
}
