package handler

import (
	"gemini-wrapper/model"
	"gemini-wrapper/service/gemini/gemini_impl"
	"net/http"

	"github.com/labstack/echo/v4"
)

type GeminiHandler struct {
	service *gemini_impl.GeminiService
}

func NewGeminiHandler(service *gemini_impl.GeminiService) *GeminiHandler {
	return &GeminiHandler{service: service}
}

// HandleAsk handles POST /api/ask.
func (g *GeminiHandler) HandleAsk(c echo.Context) error {
	if g == nil || g.service == nil {
		return c.JSON(http.StatusInternalServerError, model.AskResponse{Error: "service not initialized"})
	}

	req := new(model.AskRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, model.AskResponse{Error: "Invalid request format"})
	}

	if req.Question == "" {
		return c.JSON(http.StatusBadRequest, model.AskResponse{Error: "Question is required"})
	}

	answer, status, err := g.service.Ask(req.Question, req.Model)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.AskResponse{Error: err.Error(), Status: status})
	}

	return c.JSON(http.StatusOK, model.AskResponse{Answer: answer, Status: status})
}

// HandleGeminiAPI handles POST /v1beta/models/:model.
func (g *GeminiHandler) HandleGeminiAPI(c echo.Context) error {
	if g == nil || g.service == nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "service not initialized",
				"code":    500,
			},
		})
	}

	modelName := c.Param("model")

	var req model.GeminiAPIRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid request body",
				"code":    400,
			},
		})
	}

	if len(req.Contents) == 0 || len(req.Contents[0].Parts) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "contents[0].parts[0].text is required",
				"code":    400,
			},
		})
	}

	question := req.Contents[0].Parts[0].Text
	if question == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "text content cannot be empty",
				"code":    400,
			},
		})
	}

	answer, status, err := g.service.Ask(question, modelName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]interface{}{
				"message": err.Error(),
				"code":    500,
			},
		})
	}

	response := model.GeminiAPIResponse{
		Model:  modelName,
		Status: status,
		Candidates: []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}{
			{
				Content: struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				}{
					Parts: []struct {
						Text string `json:"text"`
					}{
						{Text: answer},
					},
				},
			},
		},
	}

	return c.JSON(http.StatusOK, response)
}
