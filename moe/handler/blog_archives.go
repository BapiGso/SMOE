package handler

import (
	"SMOE/moe/database"
	"github.com/labstack/echo/v5"
)

func Archives(c *echo.Context) error {
	qpu := new(database.QPU)
	if err := database.DB.Select(&qpu.Contents, `
		SELECT * FROM smoe_contents 
		WHERE type='post'
		AND status=?
		ORDER BY ROWID DESC `, "publish"); err != nil {
		return err
	}
	return c.Render(200, "page-timeline.template", qpu)
}
