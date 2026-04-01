package handler

import (
	"net/http"

	"gemini-wrapper/model"
	"gemini-wrapper/service/openai"

	"github.com/labstack/echo/v4"
)

type OpenAIHandler struct {
	service openai.Service
}

func NewOpenAIHandler(service openai.Service) *OpenAIHandler {
	return &OpenAIHandler{service: service}
}

func (h *OpenAIHandler) ListModels(c echo.Context) error {
	if h == nil || h.service == nil {
		return writeOpenAIError(c, &openai.APIError{HTTPStatus: 500, Type: "server_error", Code: "backend_unavailable", Message: "OpenAI adapter is not initialized"})
	}
	return c.JSON(http.StatusOK, h.service.ListModels())
}

func (h *OpenAIHandler) CreateChatCompletion(c echo.Context) error {
	if h == nil || h.service == nil {
		return writeOpenAIError(c, &openai.APIError{HTTPStatus: 500, Type: "server_error", Code: "backend_unavailable", Message: "OpenAI adapter is not initialized"})
	}

	var req model.OpenAIChatCompletionRequest
	if err := c.Bind(&req); err != nil {
		return writeOpenAIError(c, &openai.APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "invalid_json", Message: "Invalid JSON body"})
	}

	resp, err := h.service.CreateChatCompletion(req)
	if err != nil {
		return writeOpenAIError(c, err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *OpenAIHandler) CreateCompletion(c echo.Context) error {
	if h == nil || h.service == nil {
		return writeOpenAIError(c, &openai.APIError{HTTPStatus: 500, Type: "server_error", Code: "backend_unavailable", Message: "OpenAI adapter is not initialized"})
	}

	var req model.OpenAICompletionRequest
	if err := c.Bind(&req); err != nil {
		return writeOpenAIError(c, &openai.APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "invalid_json", Message: "Invalid JSON body"})
	}

	resp, err := h.service.CreateCompletion(req)
	if err != nil {
		return writeOpenAIError(c, err)
	}
	return c.JSON(http.StatusOK, resp)
}

func writeOpenAIError(c echo.Context, err error) error {
	if apiErr, ok := err.(*openai.APIError); ok {
		status := apiErr.HTTPStatus
		if status <= 0 {
			status = http.StatusInternalServerError
		}
		errType := apiErr.Type
		if errType == "" {
			errType = "server_error"
		}
		return c.JSON(status, model.OpenAIErrorResponse{Error: model.OpenAIError{
			Message: apiErr.Message,
			Type:    errType,
			Code:    apiErr.Code,
		}})
	}

	return c.JSON(http.StatusInternalServerError, model.OpenAIErrorResponse{Error: model.OpenAIError{
		Message: err.Error(),
		Type:    "server_error",
		Code:    "internal_error",
	}})
}
