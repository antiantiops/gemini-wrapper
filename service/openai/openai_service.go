package openai

import "gemini-wrapper/model"

type Service interface {
	ListModels() model.OpenAIModelListResponse
	CreateChatCompletion(req model.OpenAIChatCompletionRequest) (model.OpenAIChatCompletionResponse, error)
	CreateCompletion(req model.OpenAICompletionRequest) (model.OpenAICompletionResponse, error)
	CreateResponse(req model.OpenAIResponseRequest) (model.OpenAIResponse, error)
}

type APIError struct {
	HTTPStatus int
	Type       string
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}
