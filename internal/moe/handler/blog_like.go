package handler

import (
	"net/http"

	"github.com/BapiGso/SMOE/internal/moe/blogdata"
	"github.com/BapiGso/SMOE/internal/moe/engine"

	"github.com/labstack/echo/v5"
)

// LikePost 给文章点赞，计数在下一次刷新时写进 data/stats.json。
func LikePost(site *engine.Engine, data *blogdata.Store) echo.HandlerFunc {
	return func(c *echo.Context) error {
		cid := c.Param("cid")
		if _, ok := site.Post(cid); !ok {
			return echo.ErrNotFound
		}
		data.Like(cid)
		return c.JSON(http.StatusOK, map[string]any{"liked": true})
	}
}
