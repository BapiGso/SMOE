package handler

import (
	"SMOE/moe/store"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
)

// IndexGorm GORM版本的首页
func IndexGorm(c *echo.Context) error {
	req := &struct {
		PageNum int `param:"num" validate:"gte=0" default:"1"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	const pageSize = 5
	posts, hasMore, err := store.GetPostsByCidDesc(pageSize, (req.PageNum-1)*pageSize)
	if err != nil {
		return err
	}

	// AJAX请求：只返回文章卡片片段，不查pages
	isAjax := !strings.Contains(c.Request().Header.Get(echo.HeaderAccept), echo.MIMETextHTML)
	if isAjax {
		c.Response().Header().Set("X-Has-More", strconv.FormatBool(hasMore))
		return c.Render(200, "index-more.template", posts)
	}

	// 首页完整渲染：额外查询独立页面用于导航栏
	pages, err := store.GetAllPages()
	if err != nil {
		return err
	}

	return c.Render(200, "index.template", struct {
		Posts   []store.Contents
		Pages   []store.Contents
		HasMore bool
	}{Posts: posts, Pages: pages, HasMore: hasMore})
}
