package openai

import (
	"errors"
	"strings"
	"testing"

	"gemini-wrapper/model"
)

type fakeGeminiService struct {
	answer string
	err    error
	status *model.GeminiStatus
}

func (f *fakeGeminiService) Ask(_ string, modelName string) (string, *model.GeminiStatus, error) {
	_ = modelName
	if f.err != nil {
		return "", &model.GeminiStatus{HTTPStatus: 500, Code: "internal_error", Message: f.err.Error()}, f.err
	}
	return f.answer, f.status, nil
}

func (f *fakeGeminiService) AskWithEnv(question string, modelName string, _ map[string]string) (string, *model.GeminiStatus, error) {
	return f.Ask(question, modelName)
}

func TestCreateChatCompletionSuccess(t *testing.T) {
	svc := &fakeGeminiService{answer: "hello"}
	adapter := NewGeminiAdapter(svc)

	resp, err := adapter.CreateChatCompletion(model.OpenAIChatCompletionRequest{
		Model: "gemini-2.5-flash",
		Messages: []model.OpenAIChatMessage{
			{Role: "user", Content: "say hi"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "hello" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestCreateCompletionError(t *testing.T) {
	svc := &fakeGeminiService{err: errors.New("boom")}
	adapter := NewGeminiAdapter(svc)

	_, err := adapter.CreateCompletion(model.OpenAICompletionRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateChatCompletionRejectsNMoreThanOne(t *testing.T) {
	svc := &fakeGeminiService{answer: "hello"}
	adapter := NewGeminiAdapter(svc)

	_, err := adapter.CreateChatCompletion(model.OpenAIChatCompletionRequest{
		Model: "gemini-2.5-flash",
		Messages: []model.OpenAIChatMessage{
			{Role: "user", Content: "say hi"},
		},
		N: 2,
	})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != 400 || apiErr.Type != "invalid_request_error" || apiErr.Code != "n_not_supported" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
}

func TestCreateChatCompletionRejectsNNegative(t *testing.T) {
	svc := &fakeGeminiService{answer: "hello"}
	adapter := NewGeminiAdapter(svc)

	_, err := adapter.CreateChatCompletion(model.OpenAIChatCompletionRequest{
		Model: "gemini-2.5-flash",
		Messages: []model.OpenAIChatMessage{
			{Role: "user", Content: "say hi"},
		},
		N: -1,
	})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != 400 || apiErr.Type != "invalid_request_error" || apiErr.Code != "n_not_supported" || apiErr.Message != "n<0 is not supported" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
}

func TestCreateCompletionRejectsNMoreThanOne(t *testing.T) {
	svc := &fakeGeminiService{answer: "hello"}
	adapter := NewGeminiAdapter(svc)

	_, err := adapter.CreateCompletion(model.OpenAICompletionRequest{Prompt: "test", N: 2})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != 400 || apiErr.Type != "invalid_request_error" || apiErr.Code != "n_not_supported" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
}

func TestCreateCompletionRejectsNNegative(t *testing.T) {
	svc := &fakeGeminiService{answer: "hello"}
	adapter := NewGeminiAdapter(svc)

	_, err := adapter.CreateCompletion(model.OpenAICompletionRequest{Prompt: "test", N: -1})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != 400 || apiErr.Type != "invalid_request_error" || apiErr.Code != "n_not_supported" || apiErr.Message != "n<0 is not supported" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
}

func TestCreateResponseSuccess(t *testing.T) {
	svc := &fakeGeminiService{answer: "hello"}
	adapter := NewGeminiAdapter(svc)

	resp, err := adapter.CreateResponse(model.OpenAIResponseRequest{Model: "gemini-2.5-flash", Input: "say hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Object != "response" || resp.OutputText != "hello" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestCreateResponseRejectsInvalidInput(t *testing.T) {
	svc := &fakeGeminiService{answer: "hello"}
	adapter := NewGeminiAdapter(svc)

	_, err := adapter.CreateResponse(model.OpenAIResponseRequest{Input: []interface{}{123}})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != 400 || apiErr.Type != "invalid_request_error" || apiErr.Code != "input_invalid" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
}

func TestCreateResponseRejectsObjectItemWithoutContentOrText(t *testing.T) {
	svc := &fakeGeminiService{answer: "hello"}
	adapter := NewGeminiAdapter(svc)

	_, err := adapter.CreateResponse(model.OpenAIResponseRequest{Input: []interface{}{map[string]interface{}{"foo": "bar"}}})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != 400 || apiErr.Type != "invalid_request_error" || apiErr.Code != "input_invalid" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
}

func TestCreateResponseSkipsUnsupportedContentArrayElements(t *testing.T) {
	svc := &fakeGeminiService{answer: "hello"}
	adapter := NewGeminiAdapter(svc)

	resp, err := adapter.CreateResponse(model.OpenAIResponseRequest{Input: []interface{}{
		map[string]interface{}{
			"content": []interface{}{"ok", 123},
		},
	}})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.OutputText != "hello" {
		t.Fatalf("unexpected output: %q", resp.OutputText)
	}
}

func TestCreateResponseRejectsContentArrayMapWithoutNonEmptyText(t *testing.T) {
	svc := &fakeGeminiService{answer: "hello"}
	adapter := NewGeminiAdapter(svc)

	_, err := adapter.CreateResponse(model.OpenAIResponseRequest{Input: []interface{}{
		map[string]interface{}{
			"content": []interface{}{map[string]interface{}{"foo": "bar"}},
		},
	}})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.HTTPStatus != 400 || apiErr.Type != "invalid_request_error" || apiErr.Code != "input_invalid" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
	if !strings.Contains(apiErr.Message, "input is required") {
		t.Fatalf("expected input-required error, got: %q", apiErr.Message)
	}
}

func TestCreateChatCompletionUsesResolvedModelFromStatus(t *testing.T) {
	svc := &fakeGeminiService{
		answer: "hello",
		status: &model.GeminiStatus{Model: "gemini-2.5-flash"},
	}
	adapter := NewGeminiAdapter(svc)

	resp, err := adapter.CreateChatCompletion(model.OpenAIChatCompletionRequest{
		Model: "gemini-3.1-pro-preview",
		Messages: []model.OpenAIChatMessage{
			{Role: "user", Content: "say hi"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Model != "gemini-2.5-flash" {
		t.Fatalf("expected fallback model in response, got %q", resp.Model)
	}
}

func TestParseToolCalls(t *testing.T) {
	raw := "```json\n{\"tool_calls\": [{\"id\": \"call_123\", \"type\": \"function\", \"function\": {\"name\": \"read_file\", \"arguments\": \"{\\\"path\\\": \\\"README.md\\\"}\"}}]}\n```"
	calls, ok := parseToolCalls(raw)
	if !ok {
		t.Fatalf("expected parseToolCalls to succeed")
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Function.Name != "read_file" {
		t.Fatalf("expected read_file, got %s", calls[0].Function.Name)
	}
	if calls[0].ID != "call_123" {
		t.Fatalf("expected call_123, got %s", calls[0].ID)
	}
}

func TestBuildPromptFromRequestWithTools(t *testing.T) {
	req := model.OpenAIChatCompletionRequest{
		Tools: []model.OpenAITool{
			{
				Type: "function",
				Function: model.OpenAIFunctionDefinition{
					Name:        "read_file",
					Description: "Read a file",
				},
			},
		},
		Messages: []model.OpenAIChatMessage{
			{
				Role:    "user",
				Content: "Please read README.md",
			},
			{
				Role: "assistant",
				ToolCalls: []model.OpenAIToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: model.OpenAIFunctionCall{
							Name:      "read_file",
							Arguments: "{\"path\":\"README.md\"}",
						},
					},
				},
			},
			{
				Role:       "tool",
				ToolCallID: "call_1",
				Content:    "# Title",
			},
		},
	}

	prompt := buildPromptFromRequest(req)
	if !strings.Contains(prompt, `"name":"read_file"`) {
		t.Fatalf("tool schema missing from prompt: %q", prompt)
	}
	if !strings.Contains(prompt, "[tool result for call_1] # Title") {
		t.Fatalf("tool result missing from prompt: %q", prompt)
	}
}

func TestParseToolCallsWithoutCodeFence(t *testing.T) {
	calls, ok := parseToolCalls(`{"tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"filePath\":\"README.md\"}"}}]}`)
	if !ok || len(calls) != 1 {
		t.Fatalf("expected one tool call, got ok=%v calls=%v", ok, calls)
	}
	if calls[0].Function.Arguments != `{"filePath":"README.md"}` {
		t.Fatalf("unexpected arguments: %q", calls[0].Function.Arguments)
	}
}

func TestParseToolChoice(t *testing.T) {
	if cfg := parseToolChoice(nil); cfg.mode != "auto" {
		t.Fatalf("expected auto, got %s", cfg.mode)
	}
	if cfg := parseToolChoice("none"); cfg.mode != "none" {
		t.Fatalf("expected none, got %s", cfg.mode)
	}
	if cfg := parseToolChoice("required"); cfg.mode != "required" {
		t.Fatalf("expected required, got %s", cfg.mode)
	}
	namedMap := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{"name": "read_file"},
	}
	if cfg := parseToolChoice(namedMap); cfg.mode != "named" || cfg.funcName != "read_file" {
		t.Fatalf("expected named read_file, got %s %s", cfg.mode, cfg.funcName)
	}
}

func TestValidateToolCalls(t *testing.T) {
	tools := []model.OpenAITool{
		{
			Type: "function",
			Function: model.OpenAIFunctionDefinition{
				Name: "read_file",
				Parameters: map[string]interface{}{
					"type": "object",
					"required": []interface{}{"filePath"},
				},
			},
		},
	}

	validCalls := []model.OpenAIToolCall{
		{
			Function: model.OpenAIFunctionCall{
				Name:      "read_file",
				Arguments: `{"filePath":"README.md"}`,
			},
		},
	}

	if err := validateToolCalls(validCalls, tools, toolChoiceConfig{mode: "auto"}); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}

	// Missing required param
	invalidCalls := []model.OpenAIToolCall{
		{
			Function: model.OpenAIFunctionCall{
				Name:      "read_file",
				Arguments: `{"other":"README.md"}`,
			},
		},
	}
	if err := validateToolCalls(invalidCalls, tools, toolChoiceConfig{mode: "auto"}); err == nil {
		t.Fatalf("expected error for missing required param")
	}

	// Unknown tool
	unknownCalls := []model.OpenAIToolCall{
		{
			Function: model.OpenAIFunctionCall{
				Name:      "write_file",
				Arguments: `{}`,
			},
		},
	}
	if err := validateToolCalls(unknownCalls, tools, toolChoiceConfig{mode: "auto"}); err == nil {
		t.Fatalf("expected error for unknown tool")
	}
}
