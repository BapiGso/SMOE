package mymiddleware

import (
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
)

// JWTKey 随机密钥；设置环境变量 SMOE_DEBUG=1 可固定为 "123" 方便调试
var JWTKey = func() []byte {
	if os.Getenv("SMOE_DEBUG") != "" {
		return []byte("123")
	}
	return []byte(strconv.Itoa(rand.Int()))
}()

func JWT() echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		Skipper: func(c *echo.Context) bool {
			restricted := strings.HasPrefix(c.Path(), "/admin/") //判断当前路径是否是受限制路径（除了登录页面以外的后台路径）
			_, err := c.Cookie("smoe_token")
			return err != nil && !restricted //如果读不到cookie且不是受限制路径就跳过
		},
		ErrorHandler: func(c *echo.Context, err error) error {
			//todo 触发错误后ip限制
			c.SetCookie(&http.Cookie{Name: "smoe_token", Expires: time.Now(), MaxAge: -1, HttpOnly: true})
			return nil
		},

		SigningKey:  JWTKey,
		TokenLookup: "cookie:smoe_token",
	})
}
