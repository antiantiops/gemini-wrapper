package appmiddleware

import (
	"net/http"
	"strings"

	"gemini-wrapper/model"

	"github.com/labstack/echo/v4"
)

type AuthConfig struct {
	APIKey string
}

func RequireBearerAuth(cfg AuthConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.APIKey == "" {
				return next(c)
			}

			authorization := c.Request().Header.Get("Authorization")
			parts := strings.SplitN(authorization, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) != cfg.APIKey {
				return c.JSON(http.StatusUnauthorized, model.OpenAIErrorResponse{Error: model.OpenAIError{
					Message: "Incorrect API key provided",
					Type:    "invalid_request_error",
					Code:    "invalid_api_key",
				}})
			}

			return next(c)
		}
	}
}
