package gemini_impl

import "testing"

func TestParseGeminiOutputParsesLastJSONObject(t *testing.T) {
	out := "log line\n{\"response\":\"hello\"}\n"
	resp, ok := parseGeminiOutput(out)
	if !ok {
		t.Fatal("expected parse success")
	}
	if resp.Response != "hello" {
		t.Fatalf("unexpected response: %q", resp.Response)
	}
}

func TestParseGeminiOutputParsesFencedJSON(t *testing.T) {
	out := "some heading\n```json\n{\"response\":\"from fence\"}\n```\n"
	resp, ok := parseGeminiOutput(out)
	if !ok {
		t.Fatal("expected parse success")
	}
	if resp.Response != "from fence" {
		t.Fatalf("unexpected response: %q", resp.Response)
	}
}

func TestParseGeminiOutputParsesEscapedJSONBlob(t *testing.T) {
	out := "\"{\\\"response\\\":\\\"escaped\\\"}\""
	resp, ok := parseGeminiOutput(out)
	if !ok {
		t.Fatal("expected parse success")
	}
	if resp.Response != "escaped" {
		t.Fatalf("unexpected response: %q", resp.Response)
	}
}

func TestParseGeminiOutputFailsForMalformedPayload(t *testing.T) {
	out := "not-json at all"
	_, ok := parseGeminiOutput(out)
	if ok {
		t.Fatal("expected parse failure")
	}
}

func TestExtractFencedJSONReturnsLastJSONFence(t *testing.T) {
	out := "```json\n{\"response\":\"first\"}\n```\ntext\n```json\n{\"response\":\"last\"}\n```"
	fenced, ok := extractFencedJSON(out)
	if !ok {
		t.Fatal("expected fenced JSON")
	}
	if fenced != "{\"response\":\"last\"}" {
		t.Fatalf("unexpected fenced JSON: %q", fenced)
	}
}
