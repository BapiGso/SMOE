package handler

import (
	"SMOE/moe/store"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
)

var viewedPosts sync.Map

func init() {
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			viewedPosts.Range(func(key, _ any) bool {
				viewedPosts.Delete(key)
				return true
			})
		}
	}()
}

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

	// 防刷浏览量：同一 IP + cid 只计一次
	cidStr := fmt.Sprintf("%d", req.Cid)
	viewKey := "view:" + c.RealIP() + ":" + cidStr
	if _, loaded := viewedPosts.LoadOrStore(viewKey, struct{}{}); !loaded {
		_ = store.IncrementViews(cidStr)
		content.Views++
	}

	if strings.Contains(c.Request().Header.Get(echo.HeaderAccept), "text/markdown") {
		return renderMarkdown(c, content)
	}
	qpu := &store.QPU{
		Contents:      []store.Contents{content},
		Comments:      comments,
		CommentGroups: store.GroupComments(comments),
	}
	return c.Render(200, "post.template", qpu)
}

func renderMarkdown(c *echo.Context, content store.Contents) error {
	md := fmt.Sprintf("# %s\n\n%s", content.Title, content.Text)
	tokens := len(md) / 4 // rough estimate
	c.Response().Header().Set(echo.HeaderContentType, "text/markdown; charset=utf-8")
	c.Response().Header().Set("X-Markdown-Tokens", fmt.Sprintf("%d", tokens))
	return c.String(200, md)
}
