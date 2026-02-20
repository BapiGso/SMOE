package handler

import (
	"SMOE/moe/store"

	"github.com/labstack/echo/v5"
)

func Archives(c *echo.Context) error {
	posts, err := store.GetAllPublishedPosts()
	if err != nil {
		return err
	}
	qpu := &store.QPU{
		Contents: posts,
	}
	return c.Render(200, "page-timeline.template", qpu)
}
