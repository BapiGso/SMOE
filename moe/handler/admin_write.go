package handler

import (
	"SMOE/moe/database"
	"github.com/labstack/echo/v5"
	"strconv"
	"time"
)

func Write(c *echo.Context) error {
	qpu := &database.QPU{}
	req := &struct {
		Cid          int    `db:"cid"     param:"cid" validate:"gte=0"`
		Title        string `db:"title"   form:"title" `
		Created      int64  `db:"created" form:"created" `
		Slug         string `db:"slug"    form:"slug" `
		Text         string `db:"text"    form:"text"`
		Type         string `db:"type"    form:"type" `
		Status       string `db:"status"  form:"status" `
		AllowComment int    `db:"allowComment" form:"allowComment"`
		AllowFeed    int    `db:"allowFeed" form:"allowFeed"`
		CoverList    string `db:"coverList" form:"coverList"`
		MusicList    string `db:"musicList" form:"musicList"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	switch c.Request().Method {
	case "GET": //渲染攥写文章页面
		if err := database.DB.Select(&qpu.Contents, `
			SELECT * FROM smoe_contents
        	WHERE cid=?`, req.Cid); err != nil {
			return err
		}
		if len(qpu.Contents) == 0 && req.Cid != 0 {
			return c.Redirect(302, "/admin/write/0") //如果没查询到已有的文章则是新建文章
		}
		return c.Render(200, "write.template", qpu.Json())
	case "POST": //新建文章的API
		if err := database.DB.Get(&req.Cid, `SELECT MAX(cid)+random()%10+11 as cid FROM smoe_contents`); err != nil {
			return err
		}
		if req.Type == "post" {
			req.Slug = strconv.Itoa(req.Cid)
		}
		req.Created = time.Now().Unix()
		if _, err := database.DB.NamedExec(`
			INSERT INTO smoe_contents 
			VALUES (:cid,0,:title,:slug,:created,:text,:type,:status,
			        :allowComment,:allowFeed,0,0,:coverList,:musicList)`, req); err != nil {
			return err
		}
		return c.NoContent(204)
	case "PUT": //更新文章的API
		if _, err := database.DB.NamedExec(`
			UPDATE smoe_contents
			SET title=:title,created=:created,slug=:slug,text=:text,status=:status,allowComment=:allowComment,allowFeed=:allowFeed,coverList=:coverList,musicList=:musicList
			WHERE cid=:cid`, req); err != nil {
			return err
		}
		return c.NoContent(204)
	case "DELETE": //todo 删除文章的API

	}

	return nil
}
