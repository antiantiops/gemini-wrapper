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
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Gemini Wrapper API",
			"status":  "running",
		})
	})

	e.POST("/api/ask", func(c echo.Context) error {
		return handleAsk(c, geminiService)
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
	Answer string `json:"answer"`
	Error  string `json:"error,omitempty"`
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
	answer, err := service.Ask(req.Question, req.Model)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, AskResponse{
			Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, AskResponse{
		Answer: answer,
	})
}
