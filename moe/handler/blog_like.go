package handler

import (
	"SMOE/moe/store"
	"sync"

	"github.com/labstack/echo/v5"
)

var rateMap sync.Map // 共享限流: "like:ip:cid" → struct{}, "comment:ip" → time.Time

func LikePost(c *echo.Context) error {
	cid := c.Param("cid")
	key := "like:" + c.RealIP() + ":" + cid

	if _, loaded := rateMap.LoadOrStore(key, struct{}{}); loaded {
		return c.JSON(200, map[string]any{"liked": false})
	}

	if err := store.IncrementLikes(cid); err != nil {
		rateMap.Delete(key)
		return err
	}
	return c.JSON(200, map[string]any{"liked": true})
}
