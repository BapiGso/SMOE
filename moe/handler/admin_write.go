package handler

import (
	"SMOE/moe/store"

	"github.com/labstack/echo/v5"
)

func Write(c *echo.Context) error {
	req := &struct {
		Cid       int    `param:"cid" validate:"gte=0"`
		Title     string `form:"title"`
		Text      string `form:"text"`
		Type      string `form:"type"`
		Status    string `form:"status"`
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
	case "GET":
		if req.Cid != 0 {
			content, err := store.GetContentByCid(req.Cid)
			if err != nil {
				return c.Redirect(302, "/admin/write/0")
			}
			qpu := &store.QPU{Contents: []store.Contents{content}}
			return c.Render(200, "write.template", qpu)
		}
		return c.Render(200, "write.template", &store.QPU{})
	case "POST":
		if _, err := store.SavePost("POST", 0, req.Title, req.Text, req.Status, req.CoverList, req.MusicList); err != nil {
			return err
		}
		return c.NoContent(204)
	case "PUT":
		if _, err := store.SavePost("PUT", req.Cid, req.Title, req.Text, req.Status, req.CoverList, req.MusicList); err != nil {
			return err
		}
		return c.NoContent(204)
	}
	return nil
}
