package handler

import (
	"SMOE/moe/mymiddleware"
	"SMOE/moe/store"

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
		Url      string `xml:"url"      form:"url"    validate:"omitempty,http_url,min=1,max=200" `
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	if mymiddleware.IsAdminLoggedIn(c) {
		req.AuthorId = 1
	}
	notification, err := store.AddComment(c.Param("cid"), req.Author, req.Mail, req.Url, req.Text, req.Parent, req.AuthorId)
	if err != nil {
		return err
	}
	mymiddleware.SetCommentNotification(c, notification)
	return c.JSON(200, nil)
}
