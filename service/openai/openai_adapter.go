package openai

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gemini-wrapper/model"
	"gemini-wrapper/service/gemini"
)

type GeminiAdapter struct {
	geminiService gemini.GeminiService
}

func NewGeminiAdapter(geminiService gemini.GeminiService) *GeminiAdapter {
	return &GeminiAdapter{geminiService: geminiService}
}

func (a *GeminiAdapter) ListModels() model.OpenAIModelListResponse {
	now := time.Now().Unix()
	return model.OpenAIModelListResponse{
		Object: "list",
		Data: []model.OpenAIModel{
			{ID: "gemini-2.5-flash", Object: "model", Created: now, OwnedBy: "google"},
			{ID: "gemini-2.5-flash-lite", Object: "model", Created: now, OwnedBy: "google"},
			{ID: "gemini-2.5-pro", Object: "model", Created: now, OwnedBy: "google"},
		},
	}
}

func (a *GeminiAdapter) CreateChatCompletion(req model.OpenAIChatCompletionRequest) (model.OpenAIChatCompletionResponse, error) {
	if a.geminiService == nil {
		return model.OpenAIChatCompletionResponse{}, &APIError{HTTPStatus: 500, Type: "server_error", Code: "backend_unavailable", Message: "Gemini backend is not initialized"}
	}
	if len(req.Messages) == 0 {
		return model.OpenAIChatCompletionResponse{}, &APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "messages_required", Message: "messages is required"}
	}
	if req.Stream {
		return model.OpenAIChatCompletionResponse{}, &APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "stream_not_supported", Message: "stream=true is not supported"}
	}
	if req.N < 0 {
		return model.OpenAIChatCompletionResponse{}, &APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "n_not_supported", Message: "n<0 is not supported"}
	}
	if req.N > 1 {
		return model.OpenAIChatCompletionResponse{}, &APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "n_not_supported", Message: "n>1 is not supported"}
	}

	modelName := req.Model
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	prompt := buildPromptFromMessages(req.Messages)
	answer, status, err := a.geminiService.Ask(prompt, modelName)
	if err != nil {
		return model.OpenAIChatCompletionResponse{}, convertGeminiError(err, status)
	}

	now := time.Now().Unix()
	promptTokens := estimateTokens(prompt)
	completionTokens := estimateTokens(answer)

	return model.OpenAIChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", now),
		Object:  "chat.completion",
		Created: now,
		Model:   modelName,
		Choices: []model.OpenAIChatCompletionChoice{
			{
				Index: 0,
				Message: model.OpenAIChatMessage{
					Role:    "assistant",
					Content: answer,
				},
				FinishReason: "stop",
			},
		},
		Usage: model.OpenAIUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
}

func (a *GeminiAdapter) CreateCompletion(req model.OpenAICompletionRequest) (model.OpenAICompletionResponse, error) {
	if a.geminiService == nil {
		return model.OpenAICompletionResponse{}, &APIError{HTTPStatus: 500, Type: "server_error", Code: "backend_unavailable", Message: "Gemini backend is not initialized"}
	}
	if req.Stream {
		return model.OpenAICompletionResponse{}, &APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "stream_not_supported", Message: "stream=true is not supported"}
	}
	if req.N < 0 {
		return model.OpenAICompletionResponse{}, &APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "n_not_supported", Message: "n<0 is not supported"}
	}
	if req.N > 1 {
		return model.OpenAICompletionResponse{}, &APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "n_not_supported", Message: "n>1 is not supported"}
	}

	prompt, err := normalizePrompt(req.Prompt)
	if err != nil {
		return model.OpenAICompletionResponse{}, &APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "prompt_invalid", Message: err.Error()}
	}

	modelName := req.Model
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	answer, status, askErr := a.geminiService.Ask(prompt, modelName)
	if askErr != nil {
		return model.OpenAICompletionResponse{}, convertGeminiError(askErr, status)
	}

	now := time.Now().Unix()
	promptTokens := estimateTokens(prompt)
	completionTokens := estimateTokens(answer)

	return model.OpenAICompletionResponse{
		ID:      fmt.Sprintf("cmpl-%d", now),
		Object:  "text_completion",
		Created: now,
		Model:   modelName,
		Choices: []model.OpenAICompletionChoice{
			{
				Text:         answer,
				Index:        0,
				Logprobs:     nil,
				FinishReason: "stop",
			},
		},
		Usage: model.OpenAIUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
}

func (a *GeminiAdapter) CreateResponse(req model.OpenAIResponseRequest) (model.OpenAIResponse, error) {
	if a.geminiService == nil {
		return model.OpenAIResponse{}, &APIError{HTTPStatus: 500, Type: "server_error", Code: "backend_unavailable", Message: "Gemini backend is not initialized"}
	}

	prompt, err := normalizeResponseInput(req.Input)
	if err != nil {
		return model.OpenAIResponse{}, &APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "input_invalid", Message: err.Error()}
	}
	if strings.TrimSpace(req.Instructions) != "" {
		prompt = strings.TrimSpace(req.Instructions) + "\n\n" + prompt
	}

	modelName := req.Model
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	answer, status, askErr := a.geminiService.Ask(prompt, modelName)
	if askErr != nil {
		return model.OpenAIResponse{}, convertGeminiError(askErr, status)
	}

	now := time.Now().Unix()
	responseID := fmt.Sprintf("resp-%d", time.Now().UnixNano())
	promptTokens := estimateTokens(prompt)
	completionTokens := estimateTokens(answer)

	return model.OpenAIResponse{
		ID:        responseID,
		Object:    "response",
		CreatedAt: now,
		Status:    "completed",
		Model:     modelName,
		Output: []model.OpenAIResponseOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []model.OpenAIResponseContent{
					{Type: "output_text", Text: answer},
				},
			},
		},
		OutputText: answer,
		Usage: model.OpenAIUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
}

func buildPromptFromMessages(messages []model.OpenAIChatMessage) string {
	parts := make([]string, 0, len(messages))
	for _, m := range messages {
		role := strings.TrimSpace(m.Role)
		if role == "" {
			role = "user"
		}
		parts = append(parts, fmt.Sprintf("%s: %s", role, strings.TrimSpace(m.Content)))
	}
	return strings.Join(parts, "\n")
}

func normalizePrompt(raw interface{}) (string, error) {
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("prompt is required")
		}
		return v, nil
	case []interface{}:
		if len(v) == 0 {
			return "", fmt.Errorf("prompt is required")
		}
		parts := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return "", fmt.Errorf("prompt array must contain only strings")
			}
			parts = append(parts, s)
		}
		return strings.Join(parts, "\n"), nil
	default:
		return "", fmt.Errorf("prompt must be a string or array of strings")
	}
}

func normalizeResponseInput(raw interface{}) (string, error) {
	switch v := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return "", fmt.Errorf("input is required")
		}
		return trimmed, nil
	case []interface{}:
		if len(v) == 0 {
			return "", fmt.Errorf("input is required")
		}
		parts := make([]string, 0, len(v))
		for _, item := range v {
			s, err := normalizeResponseInputItem(item)
			if err != nil {
				return "", err
			}
			if s != "" {
				parts = append(parts, s)
			}
		}
		if len(parts) == 0 {
			return "", fmt.Errorf("input is required")
		}
		return strings.Join(parts, "\n"), nil
	default:
		return "", fmt.Errorf("input must be a string or array")
	}
}

func normalizeResponseInputItem(item interface{}) (string, error) {
	switch t := item.(type) {
	case string:
		return strings.TrimSpace(t), nil
	case map[string]interface{}:
		if content, ok := t["content"]; ok {
			s, err := normalizeResponseInputContent(content)
			if err != nil {
				return "", err
			}
			return s, nil
		}
		if text, ok := t["text"].(string); ok {
			return strings.TrimSpace(text), nil
		}
		return "", fmt.Errorf("input object item must contain content or text")
	default:
		return "", fmt.Errorf("input array contains unsupported item type")
	}
}

func normalizeResponseInputContent(content interface{}) (string, error) {
	switch c := content.(type) {
	case string:
		return strings.TrimSpace(c), nil
	case []interface{}:
		parts := make([]string, 0, len(c))
		for i, p := range c {
			switch v := p.(type) {
			case string:
				trimmed := strings.TrimSpace(v)
				if trimmed != "" {
					parts = append(parts, trimmed)
				}
			case map[string]interface{}:
				rawText, ok := v["text"]
				if !ok {
					return "", fmt.Errorf("input content[%d] object must contain non-empty text", i)
				}
				text, ok := rawText.(string)
				if !ok || strings.TrimSpace(text) == "" {
					return "", fmt.Errorf("input content[%d] object must contain non-empty text", i)
				}
				parts = append(parts, strings.TrimSpace(text))
			default:
				return "", fmt.Errorf("input content[%d] has unsupported type %T", i, p)
			}
		}
		return strings.Join(parts, "\n"), nil
	default:
		return "", fmt.Errorf("input content has unsupported type")
	}
}

func convertGeminiError(err error, status *model.GeminiStatus) error {
	if err == nil {
		return nil
	}

	httpStatus := 500
	errType := "server_error"
	errCode := "gemini_error"
	if status != nil {
		if status.HTTPStatus > 0 {
			httpStatus = status.HTTPStatus
		}
		if status.Code != "" {
			errCode = strings.ToLower(status.Code)
		}
		if httpStatus == 429 {
			errType = "rate_limit_error"
		} else if httpStatus >= 400 && httpStatus < 500 {
			errType = "invalid_request_error"
		}
	}

	log.Printf("openai adapter upstream error: status=%d type=%s code=%s err=%v", httpStatus, errType, errCode, err)

	message := "Upstream processing error"
	if httpStatus >= 500 {
		message = "An internal service error occurred"
	}

	return &APIError{
		HTTPStatus: httpStatus,
		Type:       errType,
		Code:       errCode,
		Message:    message,
	}
}

func estimateTokens(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	return (len([]rune(trimmed)) + 3) / 4
}
