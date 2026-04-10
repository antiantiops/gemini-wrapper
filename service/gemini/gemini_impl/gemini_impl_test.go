package gemini_impl

import (
	"reflect"
	"testing"
)

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

func TestParseFallbackModelsBracketSyntax(t *testing.T) {
	got := parseFallbackModels("[gemini-2.5-flash, gemini-3.1-lite-flash]")
	want := []string{"gemini-2.5-flash", "gemini-3.1-lite-flash"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected fallback models: got=%v want=%v", got, want)
	}
}

func TestParseFallbackModelsCommaSyntaxWithQuotesAndDedup(t *testing.T) {
	got := parseFallbackModels(" 'gemini-2.5-flash' , \"gemini-2.5-flash\" , gemini-2.5-pro ")
	want := []string{"gemini-2.5-flash", "gemini-2.5-pro"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected fallback models: got=%v want=%v", got, want)
	}
}

func TestBuildAttemptModelsSkipsDuplicatePrimary(t *testing.T) {
	svc := &GeminiService{fallbackModels: []string{"gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.5-pro"}}
	got := svc.buildAttemptModels("gemini-2.5-flash")
	want := []string{"gemini-2.5-flash", "gemini-2.5-pro"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected attempt models: got=%v want=%v", got, want)
	}
}
