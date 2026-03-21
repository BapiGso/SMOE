package mymiddleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/labstack/echo/v5"

	"SMOE/moe/mymiddleware"
)

func TestBrotli(t *testing.T) {
	e := echo.New()
	e.Pre(mymiddleware.Brotli())
	e.GET("/", func(c *echo.Context) error {
		return c.String(http.StatusOK, strings.Repeat("brotli works ", 32))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(echo.HeaderAcceptEncoding, "gzip, deflate, br")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("期望状态码 %d，实际 %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get(echo.HeaderContentEncoding); got != "br" {
		t.Fatalf("期望 Content-Encoding 为 br，实际 %q", got)
	}
	if got := rec.Header().Get(echo.HeaderVary); !strings.Contains(got, echo.HeaderAcceptEncoding) {
		t.Fatalf("期望 Vary 包含 %q，实际 %q", echo.HeaderAcceptEncoding, got)
	}

	decoded, err := io.ReadAll(brotli.NewReader(rec.Body))
	if err != nil {
		t.Fatalf("读取 Brotli 响应失败: %v", err)
	}

	expected := strings.Repeat("brotli works ", 32)
	if string(decoded) != expected {
		t.Fatalf("解压后的响应内容不匹配")
	}
}
