package handler

import (
	"SMOE/moe/blogdata"
	"SMOE/moe/engine"
	"bytes"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
)

// 带指纹的资源（resources.Fingerprint 生成的 name.<sha256>.css）内容永不变化，可以长期缓存
var fingerprinted = regexp.MustCompile(`\.[0-9a-f]{64}\.[a-z]+$`)

// Front 从 Hugo 的构建产物里取前台页面。
// 地址不带尾部斜杠（/archives/123），对应产物 /archives/123/index.html；
// 请求头 Accept 含 text/markdown 时优先返回同目录的 index.md，供 AI 读取原文。
// 打开文章时记一次浏览（同一 IP 同一篇在去重周期内只算一次）。
func Front(site *engine.Engine, data *blogdata.Store) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if !site.Ready() {
			c.Response().Header().Set("Retry-After", "5")
			return echo.NewHTTPError(http.StatusServiceUnavailable, "站点正在构建")
		}
		p := path.Clean("/" + c.Request().URL.Path)
		if p == "/feed" {
			p = "/feed.xml"
		}
		if dir, id := path.Split(p); dir == "/archives/" && c.Request().Method == http.MethodGet {
			if _, ok := site.Post(id); ok {
				data.View(id, c.RealIP())
			}
		}

		var candidates []string
		if path.Ext(p) != "" {
			candidates = []string{p}
		} else {
			c.Response().Header().Add("Vary", "Accept")
			if strings.Contains(c.Request().Header.Get(echo.HeaderAccept), "text/markdown") {
				candidates = append(candidates, path.Join(p, "index.md"))
			}
			candidates = append(candidates, path.Join(p, "index.html"))
		}

		for _, name := range candidates {
			b, mod, ok := site.File(name)
			if !ok {
				continue
			}
			h := c.Response().Header()
			switch ext := path.Ext(name); ext {
			case ".md":
				h.Set(echo.HeaderContentType, "text/markdown; charset=utf-8")
				h.Set("X-Markdown-Tokens", strconv.Itoa(len(b)/4)) // 粗略估算
			case ".xml":
				if name == "/feed.xml" {
					h.Set(echo.HeaderContentType, "application/rss+xml; charset=utf-8")
				}
			default:
				if ct := mime.TypeByExtension(ext); ct != "" {
					h.Set(echo.HeaderContentType, ct)
				}
			}
			if fingerprinted.MatchString(name) {
				h.Set(echo.HeaderCacheControl, "public, max-age=31536000, immutable")
			}
			http.ServeContent(c.Response(), c.Request(), name, mod, bytes.NewReader(b))
			return nil
		}
		return echo.ErrNotFound
	}
}
