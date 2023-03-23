package admin

import (
	"github.com/BapiGso/SMOE/moe/query"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"net/http"
)

func WritePost(c echo.Context) error {
	db := c.Get("db").(*sqlx.DB)
	req := new(Param)
	if err := c.Bind(req); err != nil {
		return c.String(http.StatusBadRequest, "参数Param错误")
	}
	//if req.Cid == 0 {
	//	return c.Render(200, "write-post.template", nil)
	//}
	data := struct {
		Post []query.Contents
	}{
		query.QueryWithCid(db, req.Cid),
	}
	return c.Render(200, "write-post.template", data)

}

func WritePage(c echo.Context) error {
	db := c.Get("db").(*sqlx.DB)
	req := new(Param)
	if err := c.Bind(req); err != nil {
		return c.String(http.StatusBadRequest, "参数Param错误")
	}
	//if req.Cid == 0 {
	//	return c.Render(200, "write-page.template", nil)
	//}
	data := struct {
		Page []query.Contents
	}{
		query.QueryWithCid(db, req.Cid),
	}
	return c.Render(200, "write-page.template", data)
}