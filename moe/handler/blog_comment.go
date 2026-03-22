package handler

import (
	"SMOE/moe/mymiddleware"
	"SMOE/moe/store"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

func SubmitArticleComment(c *echo.Context) error {
	rateKey := "comment:" + c.RealIP()
	if v, ok := rateMap.Load(rateKey); ok {
		if time.Since(v.(time.Time)) < 60*time.Second {
			return echo.NewHTTPError(429, "评论太频繁，请稍后再试")
		}
	}

	req := &struct {
		Parent   uint   `xml:"parent"   form:"parent" validate:""`
		Cid      uint   `xml:"cid"      form:"cid"    validate:"required"`
		Author   string `xml:"author"   form:"author" validate:"required,min=1,max=50"`
		AuthorId uint
		Mail     string `xml:"mail"     form:"mail"   validate:"email,required,min=1,max=50"`
		Text     string `xml:"text"     form:"text"   validate:"required,min=1,max=1000"`
		Url      string `xml:"url"      form:"url"    validate:"omitempty,http_url,min=1,max=200" `
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
	notification, err := store.AddComment(c.Param("cid"), req.Author, req.Mail, req.Url, req.Text, req.Parent, req.AuthorId)
	if err != nil {
		return err
	}
	mymiddleware.SetCommentNotification(c, notification)
	rateMap.Store(rateKey, time.Now())
	return c.JSON(200, nil)
}
