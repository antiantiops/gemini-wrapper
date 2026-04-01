package openai

import (
	"errors"
	"testing"

	"gemini-wrapper/model"
)

type fakeGeminiService struct {
	answer string
	err    error
}

func (f *fakeGeminiService) Ask(_ string, modelName string) (string, *model.GeminiStatus, error) {
	_ = modelName
	if f.err != nil {
		return "", &model.GeminiStatus{HTTPStatus: 500, Code: "internal_error", Message: f.err.Error()}, f.err
	}
	return f.answer, nil, nil
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
