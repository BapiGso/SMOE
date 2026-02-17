package mymiddleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/labstack/echo/v5"
)

func TestBrotliMiddleware_ResponseIsCompressed(t *testing.T) {
	e := echo.New()
	e.Pre(Brotli())

	payload := strings.Repeat("echo-brotli-", 16)
	e.GET("/brotli", func(c *echo.Context) error {
		return c.String(http.StatusOK, payload)
	})

	req := httptest.NewRequest(http.MethodGet, "/brotli", nil)
	req.Header.Set(echo.HeaderAcceptEncoding, "gzip, br")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got=%d want=%d", rec.Code, http.StatusOK)
	}

	if got := rec.Header().Get(echo.HeaderContentEncoding); got != "br" {
		t.Fatalf("unexpected content-encoding: got=%q want=%q", got, "br")
	}

	if vary := rec.Header().Get(echo.HeaderVary); !strings.Contains(vary, echo.HeaderAcceptEncoding) {
		t.Fatalf("missing vary header: got=%q", vary)
	}

	decoded, err := io.ReadAll(brotli.NewReader(bytes.NewReader(rec.Body.Bytes())))
	if err != nil {
		t.Fatalf("decode brotli body failed: %v", err)
	}

	if got := string(decoded); got != payload {
		t.Fatalf("unexpected body after decode: got=%q want=%q", got, payload)
	}
}
