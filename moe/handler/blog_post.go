package handler

import (
	"SMOE/moe/database"
	"os/exec"

	"github.com/labstack/echo/v5"
)

func Post(c *echo.Context) error {
	qpu := new(database.QPU)
	req := &struct {
		Cid int `param:"cid" validate:"gte=0"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	if err := database.DB.Select(&qpu.Contents, `
		SELECT * FROM smoe_contents 
		WHERE cid=? AND status=?
		AND type='post'`, req.Cid, "publish"); err != nil {
		return err
	}
	if len(qpu.Contents) == 0 {
		return exec.ErrDot //todo
	}
	// 递归查询 https://www.sqlite.org/lang_with.html
	if err := database.DB.Select(&qpu.Comments, `
		WITH RECURSIVE cte AS (
		SELECT * FROM smoe_comments WHERE parent=0 AND cid=? AND status=?
		UNION ALL
		SELECT s.* FROM smoe_comments AS s, cte AS c
		WHERE s.parent = c.coid
		ORDER BY ROWID DESC--深度优先
		)
		SELECT * FROM cte;`, req.Cid, "approved"); err != nil {
		return err
	}
	return c.Render(200, "post.template", qpu)
}
