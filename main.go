package main

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize Gemini service
	geminiService := NewGeminiService()

	// Routes
	healthHandler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Gemini Wrapper API",
			"status":  "running",
		})
	}
	e.GET("/", healthHandler)
	e.HEAD("/", healthHandler) // Support HEAD for health checks

	e.POST("/api/ask", func(c echo.Context) error {
		return handleAsk(c, geminiService)
	})

	// Gemini API compatible endpoint
	e.POST("/v1beta/models/:model", func(c echo.Context) error {
		return handleGeminiAPI(c, geminiService)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	e.Logger.Fatal(e.Start(":" + port))
}

// Request and Response structures
type AskRequest struct {
	Question string `json:"question" validate:"required"`
	Model    string `json:"model,omitempty"`
}

type AskResponse struct {
	Answer string        `json:"answer"`
	Error  string        `json:"error,omitempty"`
	Status *GeminiStatus `json:"status,omitempty"`
}

// Handler for /api/ask endpoint
func handleAsk(c echo.Context, service *GeminiService) error {
	req := new(AskRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, AskResponse{
			Error: "Invalid request format",
		})
	}

	if req.Question == "" {
		return c.JSON(http.StatusBadRequest, AskResponse{
			Error: "Question is required",
		})
	}

	// Send question to Gemini CLI and get response
	answer, status, err := service.Ask(req.Question, req.Model)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, AskResponse{
			Error:  err.Error(),
			Status: status,
		})
	}

	return c.JSON(http.StatusOK, AskResponse{
		Answer: answer,
		Status: status,
	})
}

// Gemini API compatible request/response structures
type GeminiAPIRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

type GeminiAPIResponse struct {
	Model      string `json:"model"`
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Status *GeminiStatus `json:"status,omitempty"`
}

// Handler for Gemini API compatible endpoint
func handleGeminiAPI(c echo.Context, service *GeminiService) error {
	// Get model from URL path
	model := c.Param("model")

	// Parse request body
	var req GeminiAPIRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid request body",
				"code":    400,
			},
		})
	}

	// Extract question from contents.parts.text
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

	// Call Gemini service
	answer, status, err := service.Ask(question, model)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": map[string]interface{}{
				"message": err.Error(),
				"code":    500,
			},
		})
	}

	// Return response in Gemini API format
	response := GeminiAPIResponse{
		Model:  model,
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
