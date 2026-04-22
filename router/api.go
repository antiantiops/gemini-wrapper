package router

import (
	"net/http"

	"gemini-wrapper/handler"
	appmiddleware "gemini-wrapper/middleware"

	"github.com/labstack/echo/v5"
)

type API struct {
	Echo          *echo.Echo
	GeminiHandler *handler.GeminiHandler
	OpenAIHandler *handler.OpenAIHandler
	OpenAIAPIKey  string
}

func (api *API) SetupRouter() {
	healthHandler := func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Gemini Wrapper API",
			"status":  "running",
		})
	}

	api.Echo.GET("/", healthHandler)
	api.Echo.HEAD("/", healthHandler)
	api.Echo.POST("/api/ask", api.GeminiHandler.HandleAsk)
	api.Echo.POST("/v1beta/models/:model", api.GeminiHandler.HandleGeminiAPI)

	if api.OpenAIHandler != nil {
		v1 := api.Echo.Group("/v1")
		v1.Use(appmiddleware.RequireBearerAuth(appmiddleware.AuthConfig{APIKey: api.OpenAIAPIKey}))
		v1.GET("/models", api.OpenAIHandler.ListModels)
		v1.POST("/chat/completions", api.OpenAIHandler.CreateChatCompletion)
		v1.POST("/completions", api.OpenAIHandler.CreateCompletion)
		v1.POST("/responses", api.OpenAIHandler.CreateResponse)
	}
}
