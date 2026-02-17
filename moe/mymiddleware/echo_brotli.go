package mymiddleware

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const brotliScheme = "br"

type BrotliConfig struct {
	Skipper   middleware.Skipper
	Level     int
	MinLength int
}

type brotliResponseWriter struct {
	io.Writer
	http.ResponseWriter
	wroteHeader       bool
	wroteBody         bool
	minLength         int
	minLengthExceeded bool
	buffer            *bytes.Buffer
	code              int
}

func Brotli() echo.MiddlewareFunc {
	return BrotliWithConfig(BrotliConfig{})
}

func BrotliWithConfig(config BrotliConfig) echo.MiddlewareFunc {
	if config.Skipper == nil {
		config.Skipper = middleware.DefaultSkipper
	}
	if config.Level == 0 {
		config.Level = brotli.DefaultCompression
	}
	if config.Level < brotli.BestSpeed || config.Level > brotli.BestCompression {
		config.Level = brotli.DefaultCompression
	}
	if config.MinLength < 0 {
		config.MinLength = 0
	}

	pool := sync.Pool{
		New: func() any {
			return brotli.NewWriterLevel(io.Discard, config.Level)
		},
	}
	bpool := sync.Pool{
		New: func() any {
			return &bytes.Buffer{}
		},
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			res := c.Response()
			res.Header().Add(echo.HeaderVary, echo.HeaderAcceptEncoding)
			if strings.Contains(c.Request().Header.Get(echo.HeaderAcceptEncoding), brotliScheme) {
				i := pool.Get()
				w, ok := i.(*brotli.Writer)
				if !ok {
					return echo.NewHTTPError(http.StatusInternalServerError, "invalid brotli writer in pool")
				}

				rw := res
				w.Reset(rw)
				buf := bpool.Get().(*bytes.Buffer)
				buf.Reset()

				brw := &brotliResponseWriter{
					Writer:         w,
					ResponseWriter: rw,
					minLength:      config.MinLength,
					buffer:         buf,
				}
				c.SetResponse(brw)

				defer func() {
					if !brw.wroteBody {
						if res.Header().Get(echo.HeaderContentEncoding) == brotliScheme {
							res.Header().Del(echo.HeaderContentEncoding)
						}
						if brw.wroteHeader {
							rw.WriteHeader(brw.code)
						}
						c.SetResponse(rw)
						w.Reset(io.Discard)
					} else if !brw.minLengthExceeded {
						c.SetResponse(rw)
						if brw.wroteHeader {
							brw.ResponseWriter.WriteHeader(brw.code)
						}
						_, _ = brw.buffer.WriteTo(rw)
						w.Reset(io.Discard)
					}

					_ = w.Close()
					bpool.Put(buf)
					pool.Put(w)
				}()
			}

			return next(c)
		}
	}
}

func (w *brotliResponseWriter) WriteHeader(code int) {
	w.Header().Del(echo.HeaderContentLength)
	w.wroteHeader = true
	w.code = code
}

func (w *brotliResponseWriter) Write(b []byte) (int, error) {
	if w.Header().Get(echo.HeaderContentType) == "" {
		w.Header().Set(echo.HeaderContentType, http.DetectContentType(b))
	}
	w.wroteBody = true

	if !w.minLengthExceeded {
		n, err := w.buffer.Write(b)
		if w.buffer.Len() >= w.minLength {
			w.minLengthExceeded = true
			w.Header().Set(echo.HeaderContentEncoding, brotliScheme)
			if w.wroteHeader {
				w.ResponseWriter.WriteHeader(w.code)
			}
			return w.Writer.Write(w.buffer.Bytes())
		}

		return n, err
	}

	return w.Writer.Write(b)
}

func (w *brotliResponseWriter) Flush() {
	if !w.minLengthExceeded {
		w.minLengthExceeded = true
		w.Header().Set(echo.HeaderContentEncoding, brotliScheme)
		if w.wroteHeader {
			w.ResponseWriter.WriteHeader(w.code)
		}
		_, _ = w.Writer.Write(w.buffer.Bytes())
	}

	if bw, ok := w.Writer.(*brotli.Writer); ok {
		_ = bw.Flush()
	}
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *brotliResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

func (w *brotliResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *brotliResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := w.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}

	return http.ErrNotSupported
}
