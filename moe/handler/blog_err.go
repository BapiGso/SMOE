package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
)

// FrontErr 按 echo.HTTPError 的状态码选择响应。
// 404 走自定义模板，其他错误返回纯文本状态，避免误导用户。
func FrontErr(c *echo.Context, err error) {
	code := http.StatusInternalServerError
	var he *echo.HTTPError
	if errors.As(err, &he) {
		code = he.Code
	}

	if code == http.StatusNotFound {
		_ = c.Render(code, "404.template", err)
		return
	}
	_ = c.String(code, http.StatusText(code))
}
