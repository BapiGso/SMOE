package handler

import (
	"SMOE/moe/database"

	"github.com/labstack/echo/v5"
)

func Manage(c *echo.Context) error {
	qpu := &database.QPU{}
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
	switch req.Type {
	case "post":
		if err := database.DB.Select(&qpu.Contents, `
		SELECT * FROM  smoe_contents
		WHERE type='post' AND status=?
		ORDER BY ROWID DESC
		LIMIT ? OFFSET ?`, req.Status, 10+1, req.Page*10-10); err != nil {
			return err
		}
	case "page":
		if err := database.DB.Select(&qpu.Contents, `
		SELECT * FROM  smoe_contents 
		WHERE type='page'
		ORDER BY ROWID `); err != nil {
			return err
		}
	case "comment":
		if err := database.DB.Select(&qpu.Comments, `
		SELECT * FROM  smoe_comments
		WHERE status=?
		ORDER BY ROWID DESC
		LIMIT ? OFFSET ?`, req.CommStatus, 10+1, req.Page*10-10); err != nil {
			return err
		}
	}

	return c.Render(200, "manage.template", qpu)
}
