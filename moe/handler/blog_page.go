package handler

import (
	"SMOE/moe/database"

	"github.com/labstack/echo/v5"
)

func Page(c *echo.Context) error {
	qpu := new(database.QPU)
	err := database.DB.Select(&qpu.Contents, `
		SELECT * FROM  smoe_contents 
		WHERE type='page' AND slug = ?`, c.Param("page"))
	if len(qpu.Contents) == 0 {
		return echo.ErrNotFound
	}
	if err != nil {
		return err
	}
	return c.Render(200, "page.template", qpu)
}
