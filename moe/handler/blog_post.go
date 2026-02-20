package handler

import (
	"SMOE/moe/store"
	"github.com/labstack/echo/v5"
)

func Post(c *echo.Context) error {
	req := &struct {
		Cid int `param:"cid" validate:"gte=0"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	content, comments, err := store.GetPostByCid(req.Cid)
	if err != nil {
		return echo.ErrNotFound
	}
	qpu := &store.QPU{
		Contents: []store.Contents{content},
		Comments: comments,
	}
	return c.Render(200, "post.template", qpu)
}
