package handler

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gemini-wrapper/service/agy_session"

	"github.com/labstack/echo/v5"
)

type SessionHandler struct{ manager *agy_session.Manager }

type turnRequest struct {
	Content string `json:"content"`
}

func NewSessionHandler(manager *agy_session.Manager) *SessionHandler {
	return &SessionHandler{manager: manager}
}

func (h *SessionHandler) Start(c *echo.Context) error {
	id, init, err := h.manager.Start()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, map[string]any{"id": id, "event": init})
}

func (h *SessionHandler) Turn(c *echo.Context) error {
	var request turnRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
	}
	if strings.TrimSpace(request.Content) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "content is required"})
	}
	if !h.manager.Exists(c.Param("id")) {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
	}
	c.Response().Header().Set("Content-Type", "application/x-ndjson")
	c.Response().WriteHeader(http.StatusOK)
	writer := bufio.NewWriter(c.Response())
	emit := func(event agy_session.Event) error {
		line, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err = writer.Write(append(line, '\n')); err != nil {
			return err
		}
		return writer.Flush()
	}
	if err := h.manager.Turn(c.Param("id"), request.Content, emit); err != nil {
		return err
	}
	return nil
}

func (h *SessionHandler) Close(c *echo.Context) error {
	if err := h.manager.Close(c.Param("id")); err != nil {
		if errors.Is(err, agy_session.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "session not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}
