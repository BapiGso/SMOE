package handler

import (
	"SMOE/moe/store"
	"github.com/labstack/echo/v5"
)

func Page(c *echo.Context) error {
	content, err := store.GetPageBySlug(c.Param("page"))
	if err != nil {
		return echo.ErrNotFound
	}
	qpu := &store.QPU{
		Contents: []store.Contents{content},
	}
	return c.Render(200, "page.template", qpu)
}
