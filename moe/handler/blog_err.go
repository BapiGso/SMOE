package handler

import (
	"github.com/labstack/echo/v5"
)

func FrontErr(c *echo.Context, err error) {

	_ = c.Render(404, "404.template", err)
}
