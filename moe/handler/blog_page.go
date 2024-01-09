package handler

import (
	"SMOE/moe/database"
	"github.com/labstack/echo/v4"
)

func Page(c echo.Context) error {
	qpu := database.NewQPU()
	defer qpu.Free()
	err := database.DB.Select(&qpu.Contents, `
		SELECT * FROM  typecho_contents 
		WHERE type='page' AND slug = ?`, c.Param("page"))
	if len(qpu.Contents) == 0 {
		return echo.NotFoundHandler(c)
	}
	if err != nil {
		return err
	}
	return c.Render(200, "page.template", qpu)
}
