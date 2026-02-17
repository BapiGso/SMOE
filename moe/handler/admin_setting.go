package handler

import (
	"github.com/labstack/echo/v5"
)

func Setting(c *echo.Context) error {
	return c.Render(200, "setting.template", nil)
}
