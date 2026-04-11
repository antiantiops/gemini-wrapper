package gemini_impl

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"gemini-wrapper/model"
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

func TestBuildCacheKeyIncludesModel(t *testing.T) {
	svc := &GeminiService{}
	k1 := svc.buildCacheKey("hello", "gemini-a")
	k2 := svc.buildCacheKey("hello", "gemini-b")
	if k1 == k2 {
		t.Fatal("expected different cache keys for different models")
	}
}

func TestCacheSetGetAndExpire(t *testing.T) {
	svc := &GeminiService{
		cacheEnabled: true,
		cacheTTL:     20 * time.Millisecond,
		cacheMaxSize: 10,
		cache:        map[string]cacheEntry{},
	}

	key := svc.buildCacheKey("q", "m")
	status := &model.GeminiStatus{HTTPStatus: 429, Model: "gemini-x"}
	svc.setCached(key, "answer", status)

	gotAnswer, gotStatus, ok := svc.getCached(key)
	if !ok || gotAnswer != "answer" || gotStatus == nil || gotStatus.Model != "gemini-x" {
		t.Fatalf("unexpected cache hit: ok=%v answer=%q status=%#v", ok, gotAnswer, gotStatus)
	}

	// Ensure cache status is a copy, not the original pointer.
	status.Model = "changed"
	_, gotStatusAgain, ok := svc.getCached(key)
	if !ok || gotStatusAgain.Model != "gemini-x" {
		t.Fatalf("expected cached status to be immutable copy, got %#v", gotStatusAgain)
	}

	time.Sleep(30 * time.Millisecond)
	if _, _, ok := svc.getCached(key); ok {
		t.Fatal("expected expired cache entry")
	}
}

func TestSetCachedRespectsMaxSize(t *testing.T) {
	svc := &GeminiService{
		cacheEnabled: true,
		cacheTTL:     time.Minute,
		cacheMaxSize: 1,
		cache:        map[string]cacheEntry{},
	}

	svc.setCached("k1", "a1", nil)
	svc.setCached("k2", "a2", nil)
	if len(svc.cache) > 1 {
		t.Fatalf("expected cache size <= 1, got %d", len(svc.cache))
	}
}

func TestParseEnvBoolDefaultsAndTruthy(t *testing.T) {
	t.Setenv("CACHE_BOOL_TEST", "")
	if !parseEnvBool("CACHE_BOOL_TEST", true) {
		t.Fatal("expected default true")
	}
	t.Setenv("CACHE_BOOL_TEST", "yes")
	if !parseEnvBool("CACHE_BOOL_TEST", false) {
		t.Fatal("expected truthy value to be true")
	}
}

func TestDiskCacheReadThrough(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "gemini-cache.db")

	svcWriter := &GeminiService{
		cacheEnabled:     true,
		cacheTTL:         time.Minute,
		cacheMaxSize:     100,
		cache:            map[string]cacheEntry{},
		diskCacheEnabled: true,
		diskCachePath:    dbPath,
		dedupeEnabled:    false,
	}
	if err := svcWriter.initDiskCache(); err != nil {
		t.Fatalf("initDiskCache writer failed: %v", err)
	}

	key := svcWriter.buildCacheKey("disk question", "gemini-2.5-flash")
	svcWriter.setCached(key, "disk-answer", &model.GeminiStatus{Model: "gemini-2.5-flash"})
	if err := svcWriter.diskDB.Close(); err != nil {
		t.Fatalf("close writer db failed: %v", err)
	}

	svcReader := &GeminiService{
		cacheEnabled:     true,
		cacheTTL:         time.Minute,
		cacheMaxSize:     100,
		cache:            map[string]cacheEntry{},
		diskCacheEnabled: true,
		diskCachePath:    dbPath,
		dedupeEnabled:    false,
	}
	if err := svcReader.initDiskCache(); err != nil {
		t.Fatalf("initDiskCache reader failed: %v", err)
	}
	defer svcReader.diskDB.Close()

	answer, status, ok := svcReader.getCached(key)
	if !ok || answer != "disk-answer" || status == nil || status.Model != "gemini-2.5-flash" {
		t.Fatalf("unexpected disk cache read-through: ok=%v answer=%q status=%#v", ok, answer, status)
	}
	if len(svcReader.cache) != 1 {
		t.Fatalf("expected memory cache repopulated from disk, size=%d", len(svcReader.cache))
	}
}
