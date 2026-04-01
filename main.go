package main

import (
	"os"

	"gemini-wrapper/handler"
	"gemini-wrapper/router"
	"gemini-wrapper/service/gemini/gemini_impl"
	"gemini-wrapper/service/openai"

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

	// Initialize Gemini and OpenAI-compatible handlers
	geminiService := gemini_impl.NewGeminiService()
	geminiHandler := handler.NewGeminiHandler(geminiService)
	openAIAdapter := openai.NewGeminiAdapter(geminiService)
	openAIHandler := handler.NewOpenAIHandler(openAIAdapter)

	api := &router.API{
		Echo:          e,
		GeminiHandler: geminiHandler,
		OpenAIHandler: openAIHandler,
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
	}
	api.SetupRouter()

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	e.Logger.Fatal(e.Start(":" + port))
}
