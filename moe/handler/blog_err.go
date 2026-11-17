package handler

import (
	"SMOE/moe/engine"
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
)

// FrontErr 按错误携带的状态码选择响应。
// 404 用 Hugo 构建出的 404.html，其他错误返回纯文本状态，避免误导用户。
func FrontErr(site *engine.Engine) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
			return
		}
		code := http.StatusInternalServerError
		var sc echo.HTTPStatusCoder
		if errors.As(err, &sc) && sc.StatusCode() != 0 {
			code = sc.StatusCode()
		}
		if code == http.StatusNotFound {
			if b, _, ok := site.File("/404.html"); ok {
				_ = c.HTMLBlob(code, b)
				return
			}
		}
		_ = c.String(code, http.StatusText(code))
	}
}
