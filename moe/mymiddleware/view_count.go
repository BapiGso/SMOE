package mymiddleware

import (
	"SMOE/moe/store"
	"github.com/labstack/echo/v5"
	"log/slog"
)

// ViewCount todo 怎样别统计到爬虫
func ViewCount(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		go func() {
			if err := store.IncrementViews(c.Param("cid")); err != nil {
				slog.Error(err.Error())
			}
		}()
		return next(c)
	}
}
