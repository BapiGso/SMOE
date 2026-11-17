package handler

import (
	"errors"
	"net/http"

	"github.com/BapiGso/SMOE/internal/moe/blogdata"
	"github.com/BapiGso/SMOE/internal/moe/engine"
	"github.com/BapiGso/SMOE/internal/moe/mymiddleware"

	"github.com/labstack/echo/v5"
)

// SubmitArticleComment 保存一条待审核评论，并通过 CommentNotify 中间件发出带审核链接的邮件。
func SubmitArticleComment(site *engine.Engine, data *blogdata.Store, mod *Moderator) echo.HandlerFunc {
	return func(c *echo.Context) error {
		req := &struct {
			Parent uint   `form:"parent"`
			Author string `form:"author" validate:"required,min=1,max=50"`
			Mail   string `form:"mail"   validate:"email,required,min=1,max=50"`
			Text   string `form:"text"   validate:"required,min=1,max=1000"`
			Url    string `form:"url"    validate:"omitempty,http_url,min=1,max=200"`
		}{}
		if err := c.Bind(req); err != nil {
			return err
		}
		if err := c.Validate(req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		cid := c.Param("cid")
		title, ok := site.Post(cid)
		if !ok {
			return echo.ErrNotFound
		}
		cm, parent, err := data.AddComment(cid, blogdata.Comment{
			Author:  req.Author,
			Mail:    req.Mail,
			URL:     req.Url,
			Content: req.Text,
			Parent:  req.Parent,
		})
		if errors.Is(err, blogdata.ErrNoComment) {
			return echo.NewHTTPError(http.StatusBadRequest, "回复的评论不存在")
		}
		if err != nil {
			return err
		}
		base := site.BaseURL()
		mymiddleware.SetCommentNotification(c, mymiddleware.CommentNotification{
			PostTitle:  title,
			PostURL:    base + "/archives/" + cid,
			Comment:    cm,
			Parent:     parent,
			ApproveURL: mod.URL(base, cid, cm.ID, "approve"),
			SpamURL:    mod.URL(base, cid, cm.ID, "spam"),
		})
		return c.JSON(http.StatusOK, nil)
	}
}
