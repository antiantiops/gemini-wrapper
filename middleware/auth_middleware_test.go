package appmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestRequireBearerAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer test-key")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := RequireBearerAuth(AuthConfig{APIKey: "test-key"})
	h := mw(func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	if err := h(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireBearerAuthInvalidKey(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := RequireBearerAuth(AuthConfig{APIKey: "test-key"})
	h := mw(func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	_ = h(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
