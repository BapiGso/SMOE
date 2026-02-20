package handler

import (
	"SMOE/moe/store"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
)

func Write(c *echo.Context) error {
	req := &struct {
		Cid       int    `param:"cid" validate:"gte=0"`
		Title     string `form:"title" `
		Created   int64  `form:"created" `
		Slug      string `form:"slug" `
		Text      string `form:"text"`
		Type      string `form:"type" `
		Status    string `form:"status" `
		CoverList string `form:"coverList"`
		MusicList string `form:"musicList"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	switch c.Request().Method {
	case "GET": //渲染攥写文章页面
		if req.Cid != 0 {
			content, err := store.GetContentByCid(req.Cid)
			if err != nil {
				return c.Redirect(302, "/admin/write/0")
			}
			qpu := &store.QPU{Contents: []store.Contents{content}}
			return c.Render(200, "write.template", qpu.Json())
		}
		qpu := &store.QPU{}
		return c.Render(200, "write.template", qpu.Json())
	case "POST": //新建文章的API
		if req.Type == "post" {
			req.Slug = strconv.Itoa(req.Cid)
		}
		req.Created = time.Now().Unix()
		if _, err := store.SavePost("POST", req.Cid, req.Title, req.Slug, req.Text, req.Type, req.Status, req.CoverList, req.MusicList, req.Created); err != nil {
			return err
		}
		return c.NoContent(204)
	case "PUT": //更新文章的API
		if _, err := store.SavePost("PUT", req.Cid, req.Title, req.Slug, req.Text, req.Type, req.Status, req.CoverList, req.MusicList, req.Created); err != nil {
			return err
		}
		return c.NoContent(204)
	}
	return nil
}
