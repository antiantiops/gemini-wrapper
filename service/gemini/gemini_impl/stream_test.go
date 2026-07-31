package gemini_impl

import "testing"

func TestParseAgyStreamEventExtractsTextDelta(t *testing.T) {
	event, err := parseAgyStreamEvent([]byte(`{"event":"step_update","step_update":{"state":"ACTIVE","step_type":"agent_response","text_delta":"hello"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Type != "step_update" || event.Delta != "hello" {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func TestParseAgyStreamEventIgnoresMetadata(t *testing.T) {
	event, err := parseAgyStreamEvent([]byte(`{"event":"init","init":{"cwd":"/app"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Delta != "" {
		t.Fatalf("unexpected delta: %q", event.Delta)
	}
}

func TestParseAgyStreamEventRejectsMalformedJSON(t *testing.T) {
	if _, err := parseAgyStreamEvent([]byte(`not-json`)); err == nil {
		t.Fatal("expected error")
	}
}
