package mymiddleware

import (
	"SMOE/moe/store"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/webdav"
)

var davHandler = &webdav.Handler{
	Prefix:     "/webdav",
	FileSystem: webdav.Dir("usr"),
	LockSystem: webdav.NewMemLS(),
}

// WebDAV returns a Pre middleware that serves /webdav/* with HTTP Basic Auth.
// Must be registered before Brotli so WebDAV responses are not Brotli-wrapped.
func WebDAV() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !strings.HasPrefix(c.Request().URL.Path, "/webdav") {
				return next(c)
			}
			username, password, ok := c.Request().BasicAuth()
			if !ok || !davAuth(username, password) {
				c.Response().Header().Set("WWW-Authenticate", `Basic realm="SMOE"`)
				return c.String(http.StatusUnauthorized, "Unauthorized")
			}
			// no-transform: 禁止 CDN/代理压缩或修改响应体（WebDAV XML/二进制会被破坏）
			// private: WebDAV 内容不应在 CDN 层缓存
			c.Response().Header().Set("Cache-Control", "no-transform, private")
			davHandler.ServeHTTP(c.Response(), c.Request())
			return nil
		}
	}
}

func davAuth(username, password string) bool {
	user, err := store.GetUser(username)
	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil
}
