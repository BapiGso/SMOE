package mymiddleware

import (
	"bufio"
	"github.com/andybalholm/brotli"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"net"
	"net/http"
	"strings"
)

type (
	// BrotliConfig defines the config for Brotli middleware.
	BrotliConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper middleware.Skipper

		// Brotli compression level.
		// Optional. Default value -1.
		// Range 0-11 0 speed 11 best compression
		Level int `yaml:"level"`
	}

	brotliResponseWriter struct {
		io.Writer
		http.ResponseWriter
	}
)

const (
	brotliScheme = "br"
)

var (
	// DefaultBrotliConfig is the default Brotli middleware config.
	DefaultBrotliConfig = BrotliConfig{
		Skipper: middleware.DefaultSkipper,
		Level:   brotli.DefaultCompression,
	}
)

// Brotli returns a middleware which compresses HTTP response using brotli compression
// scheme.
func Brotli() echo.MiddlewareFunc {
	return BrotliWithConfig(DefaultBrotliConfig)
}

// BrotliWithConfig return Brotli middleware with config.
// See: `Brotli()`.
func BrotliWithConfig(config BrotliConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultBrotliConfig.Skipper
	}
	if config.Level == 0 {
		config.Level = DefaultBrotliConfig.Level
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			c.Response().Header().Add(echo.HeaderVary, echo.HeaderAcceptEncoding)
			if strings.Contains(c.Request().Header.Get(echo.HeaderAcceptEncoding), brotliScheme) {
				c.Response().Header().Set(echo.HeaderContentEncoding, brotliScheme) // Issue #806
				rw := c.Response().Writer

				w := brotli.NewWriterOptions(rw, brotli.WriterOptions{Quality: config.Level})

				defer func() {
					if c.Response().Size == 0 {
						if c.Response().Header().Get(echo.HeaderContentEncoding) == brotliScheme {
							c.Response().Header().Del(echo.HeaderContentEncoding)
						}
						// We have to reset response to it's pristine state when
						// nothing is written to body or error is returned.
						// See issue #424, #407.
						c.Response().Writer = rw
						w.Reset(io.Discard)
					}
					w.Close()
				}()
				grw := &brotliResponseWriter{Writer: w, ResponseWriter: rw}
				c.Response().Writer = grw
			}
			return next(c)
		}
	}
}

func (w *brotliResponseWriter) WriteHeader(code int) {
	if code == http.StatusNoContent { // Issue #489
		w.ResponseWriter.Header().Del(echo.HeaderContentEncoding)
	}
	w.Header().Del(echo.HeaderContentLength) // Issue #444
	w.ResponseWriter.WriteHeader(code)
}

func (w *brotliResponseWriter) Write(b []byte) (int, error) {
	if w.Header().Get(echo.HeaderContentType) == "" {
		w.Header().Set(echo.HeaderContentType, http.DetectContentType(b))
	}
	return w.Writer.Write(b)
}

func (w *brotliResponseWriter) Flush() {
	w.Writer.(*brotli.Writer).Flush()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *brotliResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}