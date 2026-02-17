package handler

import (
	"SMOE/moe/database"
	"strings"
	"text/template"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func SubmitArticleComment(c *echo.Context) error {
	req := &struct {
		Parent   uint   `xml:"parent"   form:"parent" validate:""`
		Cid      uint   `xml:"cid"      form:"cid"    validate:"required"`
		Author   string `xml:"author"   form:"author" validate:"required,min=1,max=50"`
		AuthorId uint
		Mail     string `xml:"mail"     form:"mail"   validate:"email,required,min=1,max=50"`
		Text     string `xml:"text"     form:"text"   validate:"required,min=1,max=1000"`
		Url      string `xml:"url"      form:"url"    validate:"omitempty,url,min=1,max=50" `
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	if !strings.HasPrefix(c.Request().Referer(), c.Request().Header.Get("Origin")+"/archives/"+c.Param("cid")) {
		return echo.NewHTTPError(400, "请从评论区提交评论")
	}
	if user, ok := c.Get("user").(*jwt.Token); ok && user.Valid {
		req.AuthorId = 1
	}
	if _, err := database.DB.Exec(`
	INSERT INTO smoe_comments
	VALUES ((SELECT MAX(coid) FROM smoe_comments)+1,
	?,?,?,?,?,?,?,?,?,'waiting',?)`,
		req.Cid, time.Now().Unix(), template.HTMLEscapeString(req.Author), req.AuthorId, req.Mail, req.Url,
		c.RealIP(), c.Request().UserAgent(), template.HTMLEscapeString(req.Text), req.Parent); err != nil {
		return err
	}
	return c.JSON(200, nil)
}
