package handler

import (
	"SMOE/moe/store"
	"github.com/labstack/echo/v5"
)

func Manage(c *echo.Context) error {
	req := &struct {
		Type       string `param:"type"       default:"post" `
		CommStatus string `query:"commstatus" default:"approved" `
		Status     string `query:"status"     default:"publish" `
		Page       int    `query:"page"       default:"1"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	qpu := &store.QPU{}
	switch req.Type {
	case "post":
		posts, err := store.GetPostsByStatus(req.Status, 10, req.Page*10-10)
		if err != nil {
			return err
		}
		qpu.Contents = posts
	case "page":
		pages, err := store.GetAllPages()
		if err != nil {
			return err
		}
		qpu.Contents = pages
	case "comment":
		comments, err := store.GetAllComments(req.CommStatus, 10, req.Page*10-10)
		if err != nil {
			return err
		}
		qpu.Comments = comments
	}

	return c.Render(200, "manage.template", qpu)
}
