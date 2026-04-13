package gemini_impl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gemini-wrapper/model"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.etcd.io/bbolt"
	"golang.org/x/sync/singleflight"
)

const askCacheBucket = "ask_cache"

type GeminiService struct {
	mu             sync.Mutex
	fallbackModels []string

	cacheEnabled bool
	cacheTTL     time.Duration
	cacheMaxSize int
	cache        map[string]cacheEntry

	diskCacheEnabled    bool
	diskCachePath       string
	diskCleanupInterval time.Duration
	diskDB              *bbolt.DB

	dedupeEnabled bool
	requestGroup  singleflight.Group
}

type cacheEntry struct {
	answer    string
	status    *model.GeminiStatus
	expiresAt time.Time
}

type diskCacheRecord struct {
	Answer        string          `json:"answer"`
	StatusJSON    json.RawMessage `json:"status_json,omitempty"`
	ExpiresAtUnix int64           `json:"expires_at_unix"`
}

type askExecutionResult struct {
	answer string
	status *model.GeminiStatus
	err    error
}

// GeminiResponse represents the JSON response from gemini CLI headless mode
type GeminiResponse struct {
	Response string `json:"response"`
	Stats    struct {
		Models map[string]struct {
			Tokens struct {
				Total int `json:"total"`
			} `json:"tokens"`
		} `json:"models"`
	} `json:"stats"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Code    int    `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

func NewGeminiService() *GeminiService {
	fallbackModels := parseFallbackModels(os.Getenv("FALLBACK_MODEL"))
	cacheEnabled := parseEnvBool("CACHE_ENABLED", true)
	cacheTTL := parseEnvSeconds("CACHE_TTL_SECONDS", 1800)
	cacheMaxSize := parseEnvInt("CACHE_MAX_ENTRIES", 5000)
	dedupeEnabled := parseEnvBool("CACHE_DEDUPE_ENABLED", true)
	diskCacheEnabled := parseEnvBool("CACHE_DISK_ENABLED", true)
	diskCachePath := strings.TrimSpace(os.Getenv("CACHE_DISK_PATH"))
	diskCleanupInterval := parseEnvSeconds("CACHE_DISK_CLEANUP_INTERVAL_SECONDS", 7*24*60*60)
	if diskCachePath == "" {
		diskCachePath = "/app/cache/gemini-cache.db"
	}

	service := &GeminiService{
		fallbackModels:      fallbackModels,
		cacheEnabled:        cacheEnabled,
		cacheTTL:            cacheTTL,
		cacheMaxSize:        cacheMaxSize,
		cache:               map[string]cacheEntry{},
		diskCacheEnabled:    diskCacheEnabled,
		diskCachePath:       diskCachePath,
		diskCleanupInterval: diskCleanupInterval,
		dedupeEnabled:       dedupeEnabled,
	}
	if err := service.initDiskCache(); err != nil {
		fmt.Printf("Warning: disk cache disabled: %v\n", err)
		service.diskCacheEnabled = false
	} else if service.diskCacheEnabled && service.diskCleanupInterval > 0 {
		go service.startDiskCleanupLoop()
	}

	fmt.Printf("Gemini service initialized (using headless mode%s)\n", formatFallbackModels(fallbackModels))
	fmt.Printf("Cache config: enabled=%t ttl=%s max_entries=%d dedupe=%t disk_enabled=%t disk_path=%s disk_cleanup_interval=%s\n", cacheEnabled, cacheTTL, cacheMaxSize, dedupeEnabled, service.diskCacheEnabled, service.diskCachePath, service.diskCleanupInterval)
	return service
}

func (s *GeminiService) initDiskCache() error {
	if !s.diskCacheEnabled {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.diskCachePath), 0o755); err != nil {
		return err
	}
	db, err := bbolt.Open(s.diskCachePath, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return err
	}
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(askCacheBucket))
		return err
	}); err != nil {
		_ = db.Close()
		return err
	}
	s.diskDB = db
	return nil
}

// Ask sends a question to Gemini CLI using headless mode and returns the response.
func (s *GeminiService) Ask(question string, modelName string) (string, *model.GeminiStatus, error) {
	question = strings.TrimSpace(question)
	cacheKey := s.buildCacheKey(question, modelName)
	if answer, status, ok := s.getCached(cacheKey); ok {
		return answer, status, nil
	}

	if !s.dedupeEnabled {
		answer, status, err := s.askWithFallback(question, modelName)
		if err == nil {
			s.setCached(cacheKey, answer, status)
		}
		return answer, status, err
	}

	resultRaw, _, _ := s.requestGroup.Do(cacheKey, func() (interface{}, error) {
		answer, status, err := s.askWithFallback(question, modelName)
		if err == nil {
			s.setCached(cacheKey, answer, status)
		}
		return askExecutionResult{answer: answer, status: status, err: err}, nil
	})

	result, ok := resultRaw.(askExecutionResult)
	if !ok {
		return "", nil, fmt.Errorf("failed to process request")
	}
	return result.answer, result.status, result.err
}

func (s *GeminiService) askWithFallback(question string, modelName string) (string, *model.GeminiStatus, error) {
	attemptModels := s.buildAttemptModels(modelName)
	if len(attemptModels) == 0 {
		attemptModels = []string{""}
	}

	var preservedAnswer string
	var preservedStatus *model.GeminiStatus
	hasPreservedSuccess := false

	for i, attemptModel := range attemptModels {
		if i == 0 {
			fmt.Printf("Processing question: %q (model: %s)\n", question, printableModel(attemptModel))
		} else {
			fmt.Printf("Retrying with fallback model (%d/%d): %s\n", i, len(attemptModels)-1, printableModel(attemptModel))
		}

		answer, status, err := s.askOnce(question, attemptModel)
		if err == nil {
			if shouldFallbackAfterSuccess(status, i, len(attemptModels)) {
				status = withStatusModel(status, attemptModel)
				preservedAnswer = answer
				preservedStatus = status
				hasPreservedSuccess = true
				fmt.Printf("Successful attempt reported 429; trying fallback model next. model=%s\n", printableModel(attemptModel))
				continue
			}
			if i > 0 {
				status = withStatusModel(status, attemptModel)
				fmt.Printf("Fallback success: using model %s\n", printableModel(attemptModel))
			}
			return answer, status, nil
		}

		status = withStatusModel(status, attemptModel)
		if i == len(attemptModels)-1 || !isRetryableModelError(err, status) {
			if hasPreservedSuccess {
				return preservedAnswer, preservedStatus, nil
			}
			return "", status, err
		}

		fmt.Printf("Primary model failed with retriable error; moving to fallback model. err=%v\n", err)
	}

	if hasPreservedSuccess {
		return preservedAnswer, preservedStatus, nil
	}
	return "", nil, fmt.Errorf("failed to process request")
}

func (s *GeminiService) buildCacheKey(question string, modelName string) string {
	normalizedModel := strings.TrimSpace(modelName)
	if normalizedModel == "" {
		normalizedModel = "auto"
	}
	sum := sha256.Sum256([]byte(normalizedModel + "\n" + strings.TrimSpace(question)))
	return hex.EncodeToString(sum[:])
}

func (s *GeminiService) getCached(key string) (string, *model.GeminiStatus, bool) {
	if !s.cacheEnabled && !s.diskCacheEnabled {
		return "", nil, false
	}

	now := time.Now()
	if s.cacheEnabled {
		s.mu.Lock()
		entry, ok := s.cache[key]
		if ok {
			if now.After(entry.expiresAt) {
				delete(s.cache, key)
			} else {
				answer := entry.answer
				status := cloneGeminiStatus(entry.status)
				s.mu.Unlock()
				return answer, status, true
			}
		}
		s.mu.Unlock()
	}

	answer, status, expiresAt, ok := s.getDiskCached(key, now)
	if !ok {
		return "", nil, false
	}

	if s.cacheEnabled {
		s.mu.Lock()
		s.cache[key] = cacheEntry{answer: answer, status: cloneGeminiStatus(status), expiresAt: expiresAt}
		s.mu.Unlock()
	}
	return answer, status, true
}

func (s *GeminiService) setCached(key, answer string, status *model.GeminiStatus) {
	if (!s.cacheEnabled && !s.diskCacheEnabled) || strings.TrimSpace(answer) == "" {
		return
	}

	expiresAt := time.Now().Add(s.cacheTTL)
	if s.cacheEnabled {
		s.mu.Lock()
		if s.cacheMaxSize > 0 && len(s.cache) >= s.cacheMaxSize {
			s.evictCacheLocked(time.Now())
		}
		s.cache[key] = cacheEntry{answer: answer, status: cloneGeminiStatus(status), expiresAt: expiresAt}
		s.mu.Unlock()
	}

	s.setDiskCached(key, answer, status, expiresAt)
}

func (s *GeminiService) evictCacheLocked(now time.Time) {
	for key, entry := range s.cache {
		if now.After(entry.expiresAt) {
			delete(s.cache, key)
		}
	}
	for s.cacheMaxSize > 0 && len(s.cache) >= s.cacheMaxSize {
		for key := range s.cache {
			delete(s.cache, key)
			break
		}
	}
}

func (s *GeminiService) getDiskCached(key string, now time.Time) (string, *model.GeminiStatus, time.Time, bool) {
	if !s.diskCacheEnabled || s.diskDB == nil {
		return "", nil, time.Time{}, false
	}

	var record diskCacheRecord
	found := false
	stale := false
	err := s.diskDB.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(askCacheBucket))
		if bucket == nil {
			return nil
		}
		raw := bucket.Get([]byte(key))
		if len(raw) == 0 {
			return nil
		}
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil
		}
		found = true
		if record.ExpiresAtUnix <= now.Unix() {
			stale = true
		}
		return nil
	})
	if err != nil || !found {
		return "", nil, time.Time{}, false
	}

	if stale {
		s.deleteDiskCacheKey(key)
		return "", nil, time.Time{}, false
	}

	var status *model.GeminiStatus
	if len(record.StatusJSON) > 0 {
		var parsed model.GeminiStatus
		if err := json.Unmarshal(record.StatusJSON, &parsed); err == nil {
			status = &parsed
		}
	}
	return record.Answer, status, time.Unix(record.ExpiresAtUnix, 0), true
}

func (s *GeminiService) setDiskCached(key, answer string, status *model.GeminiStatus, expiresAt time.Time) {
	if !s.diskCacheEnabled || s.diskDB == nil || strings.TrimSpace(answer) == "" {
		return
	}

	var statusJSON []byte
	if status != nil {
		b, err := json.Marshal(status)
		if err == nil {
			statusJSON = b
		}
	}
	record := diskCacheRecord{Answer: answer, StatusJSON: statusJSON, ExpiresAtUnix: expiresAt.Unix()}
	payload, err := json.Marshal(record)
	if err != nil {
		return
	}

	_ = s.diskDB.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(askCacheBucket))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(key), payload)
	})
}

func (s *GeminiService) deleteDiskCacheKey(key string) {
	if !s.diskCacheEnabled || s.diskDB == nil {
		return
	}
	_ = s.diskDB.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(askCacheBucket))
		if bucket == nil {
			return nil
		}
		return bucket.Delete([]byte(key))
	})
}

func (s *GeminiService) startDiskCleanupLoop() {
	ticker := time.NewTicker(s.diskCleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanupExpiredDiskCache(time.Now().Unix())
	}
}

func (s *GeminiService) cleanupExpiredDiskCache(nowUnix int64) {
	if !s.diskCacheEnabled || s.diskDB == nil {
		return
	}
	_ = s.diskDB.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(askCacheBucket))
		if bucket == nil {
			return nil
		}
		cursor := bucket.Cursor()
		for key, value := cursor.First(); key != nil; key, value = cursor.Next() {
			var record diskCacheRecord
			if err := json.Unmarshal(value, &record); err != nil || record.ExpiresAtUnix <= nowUnix {
				if err := cursor.Delete(); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func cloneGeminiStatus(status *model.GeminiStatus) *model.GeminiStatus {
	if status == nil {
		return nil
	}
	statusCopy := *status
	return &statusCopy
}

func parseEnvBool(key string, defaultValue bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return defaultValue
	}
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

func parseEnvInt(key string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}

func parseEnvSeconds(key string, defaultSeconds int) time.Duration {
	seconds := parseEnvInt(key, defaultSeconds)
	return time.Duration(seconds) * time.Second
}

func (s *GeminiService) askOnce(question string, modelName string) (string, *model.GeminiStatus, error) {
	// Prepare the command arguments
	args := []string{
		"--prompt", question,
		"--output-format", "json",
	}

	// Add model if specified
	if modelName != "" {
		args = append(args, "--model", modelName)
	}

	// Create command
	cmd := exec.Command("gemini", args...)

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"HOME=/app",
		"GEMINI_CONFIG_DIR=/app/.gemini",
		"XDG_CONFIG_HOME=/app",
	)

	// Run command and capture output
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	status := detectUpstreamStatus(outputStr, nil)
	if err != nil {
		// Provide helpful error messages for common issues
		if strings.Contains(outputStr, "ModelNotFoundError") || strings.Contains(outputStr, "not found") {
			return "", status, fmt.Errorf("model not found: the model '%s' doesn't exist or isn't available. Use 'gemini-2.5-flash', 'gemini-2.5-flash-lite', 'gemini-2.5-pro', or omit model for auto-selection", modelName)
		}

		if strings.Contains(outputStr, "authentication") || strings.Contains(outputStr, "auth") {
			return "", status, fmt.Errorf("authentication error: make sure ~/.gemini is mounted correctly and you're authenticated")
		}

		response, ok := parseGeminiOutput(outputStr)
		if ok {
			status = detectUpstreamStatus(outputStr, &response)
			if response.Error != nil {
				answer := strings.TrimSpace(response.Response)
				if status != nil && status.HTTPStatus == http.StatusTooManyRequests && answer != "" {
					return answer, status, nil
				}
				return "", status, fmt.Errorf("gemini error: %s - %s", response.Error.Type, response.Error.Message)
			}

			answer := strings.TrimSpace(response.Response)
			if answer != "" {
				return answer, status, nil
			}
		}

		return "", status, fmt.Errorf("failed to execute gemini CLI: %v (output: %s)", err, outputStr)
	}

	response, ok := parseGeminiOutput(outputStr)
	if !ok {
		// No valid JSON found, return raw output
		fmt.Printf("Warning: No valid JSON found in output\n")
		return strings.TrimSpace(outputStr), status, nil
	}

	status = detectUpstreamStatus(outputStr, &response)

	// Check for errors in response
	if response.Error != nil {
		answer := strings.TrimSpace(response.Response)
		if status != nil && status.HTTPStatus == http.StatusTooManyRequests && answer != "" {
			return answer, status, nil
		}
		errorMsg := fmt.Sprintf("gemini error: %s - %s", response.Error.Type, response.Error.Message)

		// Provide helpful message for common errors
		if strings.Contains(errorMsg, "ModelNotFoundError") || strings.Contains(errorMsg, "not found") {
			return "", status, fmt.Errorf("model not found: the specified model doesn't exist or isn't available. Try using 'gemini-2.5-flash' or don't specify a model for auto-selection")
		}

		return "", status, fmt.Errorf("%s", errorMsg)
	}

	// Return the response text
	answer := strings.TrimSpace(response.Response)
	if answer == "" {
		return "", status, fmt.Errorf("received empty response from gemini")
	}

	fmt.Printf("✓ Response received (%d chars)\n", len(answer))
	return answer, status, nil
}

// AskWithEnv sends a question with custom environment variables
func (s *GeminiService) AskWithEnv(question string, model string, _ map[string]string) (string, *model.GeminiStatus, error) {
	// For headless mode, we don't need to modify process env vars
	// Just pass them directly to the command
	return s.Ask(question, model)
}

func parseGeminiOutput(outputStr string) (GeminiResponse, bool) {
	candidates := buildParseCandidates(outputStr)
	attemptErrors := make([]string, 0, len(candidates))

	for _, candidate := range candidates {
		response, err := tryParseGeminiResponse(candidate.payload)
		if err == nil {
			return response, true
		}
		attemptErrors = append(attemptErrors, fmt.Sprintf("%s: %v", candidate.name, err))
	}

	if len(attemptErrors) > 0 {
		fmt.Printf("Warning: Failed to parse JSON response. attempts=%s\n", strings.Join(attemptErrors, " | "))
	}
	return GeminiResponse{}, false
}

type parseCandidate struct {
	name    string
	payload string
}

func buildParseCandidates(outputStr string) []parseCandidate {
	trimmed := strings.TrimSpace(outputStr)
	if trimmed == "" {
		return nil
	}

	candidates := make([]parseCandidate, 0, 3)
	seen := map[string]struct{}{}
	add := func(name, payload string) {
		payload = strings.TrimSpace(payload)
		if payload == "" {
			return
		}
		if _, ok := seen[payload]; ok {
			return
		}
		seen[payload] = struct{}{}
		candidates = append(candidates, parseCandidate{name: name, payload: payload})
	}

	add("full_output", trimmed)
	if extracted, ok := extractLastJSONObject(trimmed); ok {
		add("last_json_object", extracted)
	}
	if fenced, ok := extractFencedJSON(trimmed); ok {
		add("fenced_json", fenced)
	}

	return candidates
}

func tryParseGeminiResponse(payload string) (GeminiResponse, error) {
	var response GeminiResponse
	if err := json.Unmarshal([]byte(payload), &response); err == nil {
		return response, nil
	}

	var encoded string
	if err := json.Unmarshal([]byte(payload), &encoded); err != nil {
		return GeminiResponse{}, err
	}

	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return GeminiResponse{}, fmt.Errorf("decoded payload is empty")
	}
	if err := json.Unmarshal([]byte(encoded), &response); err != nil {
		return GeminiResponse{}, err
	}
	return response, nil
}

func parseFallbackModels(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		raw = strings.TrimSpace(raw[1 : len(raw)-1])
	}

	parts := strings.Split(raw, ",")
	fallbacks := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		candidate := strings.TrimSpace(p)
		candidate = strings.Trim(candidate, "\"'")
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		fallbacks = append(fallbacks, candidate)
	}
	return fallbacks
}

func (s *GeminiService) buildAttemptModels(primary string) []string {
	attempts := make([]string, 0, 1+len(s.fallbackModels))
	attempts = append(attempts, strings.TrimSpace(primary))
	seen := map[string]struct{}{attempts[0]: {}}
	for _, fallback := range s.fallbackModels {
		fallback = strings.TrimSpace(fallback)
		if fallback == "" {
			continue
		}
		if _, ok := seen[fallback]; ok {
			continue
		}
		seen[fallback] = struct{}{}
		attempts = append(attempts, fallback)
	}
	return attempts
}

func withStatusModel(status *model.GeminiStatus, modelName string) *model.GeminiStatus {
	if strings.TrimSpace(modelName) == "" {
		return status
	}
	if status == nil {
		return &model.GeminiStatus{Model: modelName}
	}
	status.Model = modelName
	return status
}

func printableModel(modelName string) string {
	if strings.TrimSpace(modelName) == "" {
		return "auto"
	}
	return modelName
}

func isRetryableModelError(err error, status *model.GeminiStatus) bool {
	if status != nil && status.HTTPStatus == http.StatusTooManyRequests {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "rate limit") ||
		strings.Contains(message, "ratelimit") ||
		strings.Contains(message, "resource_exhausted") ||
		strings.Contains(message, "capacity") ||
		strings.Contains(message, "quota")
}

func detectUpstreamStatus(outputStr string, response *GeminiResponse) *model.GeminiStatus {
	if inferred := detectRateLimitStatus(outputStr); inferred != nil {
		return inferred
	}

	if response != nil && response.Error != nil {
		status := &model.GeminiStatus{Message: response.Error.Message}
		if response.Error.Type != "" {
			status.Code = response.Error.Type
		}
		if response.Error.Code >= 100 && response.Error.Code <= 599 {
			status.HTTPStatus = response.Error.Code
		} else if parsed, ok := parseHTTPStatusFromCode(response.Error.Type); ok {
			status.HTTPStatus = parsed
		}
		if status.HTTPStatus != 0 || status.Code != "" || status.Message != "" {
			return status
		}
	}

	return nil
}

func detectRateLimitStatus(outputStr string) *model.GeminiStatus {
	lower := strings.ToLower(outputStr)

	if strings.Contains(outputStr, "\"code\": 429") ||
		strings.Contains(outputStr, "\"status\": 429") ||
		strings.Contains(outputStr, "status 429") ||
		strings.Contains(outputStr, "HTTP/1.1 429") ||
		strings.Contains(outputStr, "HTTP/2 429") ||
		strings.Contains(outputStr, "Too Many Requests") ||
		strings.Contains(outputStr, "rateLimitExceeded") ||
		strings.Contains(outputStr, "RESOURCE_EXHAUSTED") {
		return &model.GeminiStatus{
			HTTPStatus: http.StatusTooManyRequests,
			Code:       "RESOURCE_EXHAUSTED",
			Message:    "Upstream rate limited or model capacity exhausted",
		}
	}

	// Require stronger contextual phrases to avoid classifying ordinary text as 429.
	if strings.Contains(lower, "quota exceeded") ||
		strings.Contains(lower, "exceeded quota") ||
		strings.Contains(lower, "capacity exceeded") ||
		strings.Contains(lower, "exceeded capacity") ||
		strings.Contains(lower, "rate limit exceeded") {
		return &model.GeminiStatus{
			HTTPStatus: http.StatusTooManyRequests,
			Code:       "RESOURCE_EXHAUSTED",
			Message:    "Upstream rate limited or model capacity exhausted",
		}
	}

	errorContext := strings.Contains(lower, "\"error\"") || strings.Contains(lower, "error:") || strings.Contains(lower, "\"headers\"")
	if errorContext && hasAnyWord(lower, "quota", "capacity") && hasAnyWord(lower, "rate", "limit", "exceeded", "exhausted") {
		return &model.GeminiStatus{
			HTTPStatus: http.StatusTooManyRequests,
			Code:       "RESOURCE_EXHAUSTED",
			Message:    "Upstream rate limited or model capacity exhausted",
		}
	}

	return nil
}

func hasAnyWord(input string, words ...string) bool {
	if len(words) == 0 {
		return false
	}
	set := map[string]struct{}{}
	for _, token := range tokenizeLower(input) {
		set[token] = struct{}{}
	}
	for _, w := range words {
		if _, ok := set[strings.ToLower(strings.TrimSpace(w))]; ok {
			return true
		}
	}
	return false
}

func tokenizeLower(input string) []string {
	normalized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return ' '
	}, input)
	return strings.Fields(normalized)
}

func parseHTTPStatusFromCode(code string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(code))
	if err != nil {
		return 0, false
	}
	if parsed < 100 || parsed > 599 {
		return 0, false
	}
	return parsed, true
}

func extractLastJSONObject(outputStr string) (string, bool) {
	depth := 0
	inString := false
	escaped := false
	end := -1

	// Scan backwards to find the last complete JSON object while ignoring braces in strings.
	for i := len(outputStr) - 1; i >= 0; i-- {
		ch := outputStr[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			continue
		}

		if ch == '}' {
			if end == -1 {
				end = i
			}
			depth++
			continue
		}

		if ch == '{' && end != -1 {
			depth--
			if depth == 0 {
				return outputStr[i : end+1], true
			}
		}
	}

	return "", false
}

func extractFencedJSON(outputStr string) (string, bool) {
	last := ""
	for i := 0; i < len(outputStr); {
		startRel := strings.Index(outputStr[i:], "```")
		if startRel == -1 {
			break
		}
		start := i + startRel

		headerStart := start + 3
		lineRel := strings.IndexByte(outputStr[headerStart:], '\n')
		if lineRel == -1 {
			break
		}
		lineEnd := headerStart + lineRel
		language := strings.TrimSpace(outputStr[headerStart:lineEnd])

		contentStart := lineEnd + 1
		closeRel := strings.Index(outputStr[contentStart:], "```")
		if closeRel == -1 {
			break
		}
		contentEnd := contentStart + closeRel
		content := strings.TrimSpace(outputStr[contentStart:contentEnd])

		lowerLanguage := strings.ToLower(language)
		if content != "" && (lowerLanguage == "json" || lowerLanguage == "" || strings.HasPrefix(lowerLanguage, "json ")) {
			last = content
		}

		i = contentEnd + 3
	}

	if last == "" {
		return "", false
	}
	return last, true
}

func shouldFallbackAfterSuccess(status *model.GeminiStatus, attemptIndex int, totalAttempts int) bool {
	if status == nil || status.HTTPStatus != http.StatusTooManyRequests {
		return false
	}
	return attemptIndex < totalAttempts-1
}

func formatFallbackModels(fallbackModels []string) string {
	if len(fallbackModels) == 0 {
		return ""
	}
	return fmt.Sprintf(", fallback models: %s", strings.Join(fallbackModels, ", "))
}