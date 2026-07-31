package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gemini-wrapper/model"
	"gemini-wrapper/service/openai"

	"github.com/labstack/echo/v5"
)

type OpenAIHandler struct {
	service openai.Service
}

func NewOpenAIHandler(service openai.Service) *OpenAIHandler {
	return &OpenAIHandler{service: service}
}

func (h *OpenAIHandler) ListModels(c *echo.Context) error {
	if h == nil || h.service == nil {
		return writeOpenAIError(c, &openai.APIError{HTTPStatus: 500, Type: "server_error", Code: "backend_unavailable", Message: "OpenAI adapter is not initialized"})
	}
	return c.JSON(http.StatusOK, h.service.ListModels())
}

func (h *OpenAIHandler) CreateChatCompletion(c *echo.Context) error {
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

func (h *OpenAIHandler) CreateCompletion(c *echo.Context) error {
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

func (h *OpenAIHandler) CreateResponse(c *echo.Context) error {
	if h == nil || h.service == nil {
		return writeOpenAIError(c, &openai.APIError{HTTPStatus: 500, Type: "server_error", Code: "backend_unavailable", Message: "OpenAI adapter is not initialized"})
	}

	var req model.OpenAIResponseRequest
	if err := c.Bind(&req); err != nil {
		return writeOpenAIError(c, &openai.APIError{HTTPStatus: 400, Type: "invalid_request_error", Code: "invalid_json", Message: "Invalid JSON body"})
	}

	if req.Stream {
		return h.streamResponse(c, req)
	}
	resp, err := h.service.CreateResponse(req)
	if err != nil {
		return writeOpenAIError(c, err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *OpenAIHandler) streamResponse(c *echo.Context, req model.OpenAIResponseRequest) error {
	r := c.Response()
	r.Header().Set(echo.HeaderContentType, "text/event-stream")
	r.Header().Set("Cache-Control", "no-cache")
	r.Header().Set("Connection", "keep-alive")
	r.WriteHeader(http.StatusOK)
	flusher, ok := r.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not implement http.Flusher")
	}
	writeEvent := func(event string, payload interface{}) error {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(r, "event: %s\ndata: %s\n\n", event, body); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}
	if err := writeEvent("response.created", map[string]interface{}{"type": "response.created", "response": map[string]interface{}{"object": "response", "status": "in_progress"}}); err != nil {
		return err
	}
	var output string
	resp, err := h.service.StreamResponse(c.Request().Context(), req, func(delta string) error {
		output += delta
		return writeEvent("response.output_text.delta", map[string]interface{}{"type": "response.output_text.delta", "delta": delta})
	})
	if err != nil {
		_ = writeEvent("error", model.OpenAIErrorResponse{Error: model.OpenAIError{Message: err.Error(), Type: "server_error", Code: "stream_error"}})
		return nil
	}
	if err := writeEvent("response.output_text.done", map[string]interface{}{"type": "response.output_text.done", "text": output}); err != nil {
		return err
	}
	if err := writeEvent("response.completed", map[string]interface{}{"type": "response.completed", "response": resp}); err != nil {
		return err
	}
	_, err = fmt.Fprint(r, "data: [DONE]\n\n")
	flusher.Flush()
	return err
}

// writeResponseSSE preserves the former buffered SSE helper for compatibility.
func writeResponseSSE(c *echo.Context, resp model.OpenAIResponse) error {
	r := c.Response()
	r.Header().Set(echo.HeaderContentType, "text/event-stream")
	r.Header().Set("Cache-Control", "no-cache")
	r.Header().Set("Connection", "keep-alive")
	r.WriteHeader(http.StatusOK)
	flusher, ok := r.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not implement http.Flusher")
	}

	writeEvent := func(event string, payload interface{}) error {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(r, "event: %s\ndata: %s\n\n", event, string(body)); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := writeEvent("response.created", map[string]interface{}{"type": "response.created", "response": resp}); err != nil {
		return err
	}
	if resp.OutputText != "" {
		if err := writeEvent("response.output_text.delta", map[string]interface{}{"type": "response.output_text.delta", "delta": resp.OutputText}); err != nil {
			return err
		}
		if err := writeEvent("response.output_text.done", map[string]interface{}{"type": "response.output_text.done", "text": resp.OutputText}); err != nil {
			return err
		}
	}
	if err := writeEvent("response.completed", map[string]interface{}{"type": "response.completed", "response": resp}); err != nil {
		return err
	}

	_, err := fmt.Fprint(r, "data: [DONE]\n\n")
	flusher.Flush()
	return err
}

func writeOpenAIError(c *echo.Context, err error) error {
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
