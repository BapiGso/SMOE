package handler

import (
	"SMOE/moe/blogdata"
	"SMOE/moe/engine"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
)

// 审核链接的有效期
const moderateTTL = 30 * 24 * time.Hour

// 链接里的动作 → 评论状态及页面文案
var moderateActions = map[string]struct{ status, verb string }{
	"approve": {"approved", "通过"},
	"spam":    {"spam", "标为垃圾"},
}

// Moderator 生成并校验邮件里的评论审核链接。链接带 HMAC 签名，本身就是凭证，不需要登录。
type Moderator struct {
	key []byte
}

// NewModerator 用 secret（配置里的密码哈希）派生签名密钥；改密码会让旧链接失效。
// secret 为空时用随机密钥，重启后旧链接失效。
func NewModerator(secret string) *Moderator {
	if secret == "" {
		slog.Warn("no password in config, comment moderation links expire on restart")
		secret = rand.Text()
	}
	sum := sha256.Sum256([]byte("smoe-comment-moderation\x00" + secret))
	return &Moderator{key: sum[:]}
}

func (m *Moderator) sign(post string, id uint, action string, exp int64) string {
	mac := hmac.New(sha256.New, m.key)
	fmt.Fprintf(mac, "%s\x00%d\x00%s\x00%d", post, id, action, exp)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// URL 返回某条评论的审核链接，base 是站点地址（不带尾部斜杠）。
func (m *Moderator) URL(base, post string, id uint, action string) string {
	exp := time.Now().Add(moderateTTL).Unix()
	q := url.Values{}
	q.Set("post", post)
	q.Set("id", strconv.FormatUint(uint64(id), 10))
	q.Set("action", action)
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", m.sign(post, id, action, exp))
	return base + "/comment/moderate?" + q.Encode()
}

func (m *Moderator) verify(c *echo.Context) (post string, id uint, action string, err error) {
	post, action = c.FormValue("post"), c.FormValue("action")
	n, err1 := strconv.ParseUint(c.FormValue("id"), 10, 32)
	exp, err2 := strconv.ParseInt(c.FormValue("exp"), 10, 64)
	if err1 != nil || err2 != nil {
		return "", 0, "", errors.New("链接不完整")
	}
	if !hmac.Equal([]byte(c.FormValue("sig")), []byte(m.sign(post, uint(n), action, exp))) {
		return "", 0, "", errors.New("链接无效")
	}
	if time.Now().Unix() > exp {
		return "", 0, "", errors.New("链接已过期，请在 usr/data/comments 里手动修改")
	}
	if _, ok := moderateActions[action]; !ok {
		return "", 0, "", errors.New("未知操作")
	}
	return post, uint(n), action, nil
}

var moderatePage = template.Must(template.New("moderate").Parse(`<!DOCTYPE html>
<html lang="zh-Hans">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="robots" content="noindex">
<title>评论审核</title>
<style>
body{font:16px/1.7 system-ui,sans-serif;max-width:36em;margin:3em auto;padding:0 1em;color:#333}
blockquote{margin:1em 0;padding:.6em 1em;border-left:4px solid #9370DB;background:#f6f4fa;white-space:pre-wrap;word-break:break-word}
button{font:inherit;padding:.4em 1.6em;cursor:pointer}
small{color:#888}
</style>
</head>
<body>
{{- if .Error }}
<p>{{ .Error }}</p>
{{- else }}
<h1>{{ .Heading }}</h1>
<p>文章：<a href="{{ .PostURL }}">{{ .PostTitle }}</a></p>
<p>{{ .C.Author }} &lt;{{ .C.Mail }}&gt;{{ with .C.URL }} · {{ . }}{{ end }}<br><small>{{ .C.Created }} · 当前状态 {{ .C.Status }}</small></p>
<blockquote>{{ .C.Content }}</blockquote>
{{- if .Confirm }}
<form method="post"><button type="submit">确认{{ .Verb }}</button></form>
{{- else }}
<p>页面会在几秒内更新。</p>
{{- end }}
{{- end }}
</body>
</html>
`))

// Handler 处理审核链接：GET 只显示确认页（邮件客户端会预取链接，不能直接生效），POST 才执行。
func (m *Moderator) Handler(site *engine.Engine, data *blogdata.Store) echo.HandlerFunc {
	return func(c *echo.Context) error {
		render := func(code int, v map[string]any) error {
			c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
			c.Response().WriteHeader(code)
			return moderatePage.Execute(c.Response(), v)
		}
		post, id, action, err := m.verify(c)
		if err != nil {
			return render(http.StatusForbidden, map[string]any{"Error": err.Error()})
		}
		act := moderateActions[action]

		var cm blogdata.Comment
		if c.Request().Method == http.MethodPost {
			cm, err = data.SetCommentStatus(post, id, act.status)
		} else {
			cm, err = data.Comment(post, id)
		}
		if err != nil {
			return render(http.StatusNotFound, map[string]any{"Error": "评论不存在：" + err.Error()})
		}

		title, _ := site.Post(post)
		v := map[string]any{
			"C":         cm,
			"Verb":      act.verb,
			"PostTitle": title,
			"PostURL":   "/archives/" + post + "#" + strconv.FormatUint(uint64(id), 10),
			"Confirm":   c.Request().Method != http.MethodPost,
			"Heading":   act.verb + "这条评论？",
		}
		if c.Request().Method == http.MethodPost {
			v["Heading"] = "已" + act.verb
		}
		return render(http.StatusOK, v)
	}
}
