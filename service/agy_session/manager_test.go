package agy_session

import (
	"os/exec"
	"strings"
	"testing"
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
	// shell receives fixed test NDJSON through environment, not user input.
	t.Setenv("EVENTS", events)
	manager := NewManagerWithFactory(fakeCommand(events))
	id, init, err := manager.Start()
	if err != nil {
		t.Fatal(err)
	}
	if init["conversation_id"] != "agy-1" {
		t.Fatalf("init=%v", init)
	}
	var got []Event
	if err := manager.Turn(id, "list files", func(event Event) error { got = append(got, event); return nil }); err != nil {
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
	err := manager.Turn("missing", "hello", func(Event) error { return nil })
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
