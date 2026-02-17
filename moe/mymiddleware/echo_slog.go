package mymiddleware

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

var slogDefault = func() *slog.Logger {
	l := slog.New(
		slog.NewTextHandler(
			os.Stdout, &slog.HandlerOptions{
				AddSource: false,
				Level:     slog.LevelInfo,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			},
		),
	)
	slog.SetDefault(l)
	return slog.Default()
}()

func Slog() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:   true,
		LogURI:      true,
		LogStatus:   true,
		LogRemoteIP: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			slog.LogAttrs(context.Background(), slog.LevelInfo, fmt.Sprintf("err=%v", v.Error),
				slog.String("Method", v.Method),
				slog.String("IP", v.RemoteIP),
				slog.Int("Status", v.Status),
				slog.String("Url", v.URI),
			)
			return nil
		},
	})
}
