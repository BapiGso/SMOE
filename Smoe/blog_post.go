package Smoe

import (
	"github.com/labstack/echo/v4"
	"main/smoe/query"
	"net/http"
)

func (s *Smoe) Post(c echo.Context) error {
	cid, _ := IsNum(c.Param("cid"))
	data := struct {
		Post     []query.Contents
		TestPost query.Contents
	}{
		query.QueryWithCid(s.Db, cid),
		query.TestQueryPostWithCid(s.Db, cid),
	}
	return c.Render(http.StatusOK, "testpost.template", data)
}
