package handler

import (
	"SMOE/moe/database"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
	"strings"
)

// Index TODO 加载更多。ajax
func Index(c *echo.Context) error {
	qpu := new(database.QPU)
	req := &struct {
		PageNum int `param:"num" validate:"gte=1" default:"1"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	const postsPerPage = 5
	if err := qpu.LoadIndexContents("publish", req.PageNum, postsPerPage); err != nil {
		return err
	}

	isAjax := !strings.Contains(c.Request().Header.Get(echo.HeaderAccept), echo.MIMETextHTML)
	data := struct {
		*database.QPU
		IsAjax bool
	}{
		QPU:    qpu,
		IsAjax: isAjax,
	}

	return c.Render(200, "index.template", data)
}

// Deprecated: use x-request-with instead of
func IndexAjax(c *echo.Context) error {
	_ = c.Get("db").(*sqlx.DB)
	return c.Render(200, "index.template", struct {
		*database.QPU
		IsAjax bool
	}{
		QPU:    new(database.QPU),
		IsAjax: true,
	})
}
