package handler

import (
	"SMOE/moe/store"

	"github.com/labstack/echo/v5"
)

func LikePost(c *echo.Context) error {
	cid := c.Param("cid")
	if err := store.IncrementLikes(cid); err != nil {
		return err
	}
	return c.JSON(200, map[string]any{"liked": true})
}
