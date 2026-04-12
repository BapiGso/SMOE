package mymiddleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"SMOE/moe/mymiddleware"

	"github.com/labstack/echo/v5"
)

func TestSameOriginOnly(t *testing.T) {
	e := echo.New()
	e.POST("/comment", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	}, mymiddleware.SameOriginOnly())

	t.Run("allow_origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/comment", nil)
		req.Host = "example.com"
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
		}
	})

	t.Run("allow_referer_fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/comment", nil)
		req.Host = "example.com"
		req.Header.Set("Referer", "http://example.com/archives/1")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
		}
	})

	t.Run("reject_cross_site", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/comment", nil)
		req.Host = "example.com"
		req.Header.Set("Origin", "http://evil.example")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}
