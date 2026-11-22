package moe

import (
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

	"github.com/BapiGso/SMOE/internal/moe/blogdata"
)

// 审核链接的有效期
const moderateTTL = 30 * 24 * time.Hour

// 链接里的动作 → 评论状态及页面文案
var moderateActions = map[string]struct{ status, verb string }{
	"approve": {"approved", "通过"},
	"spam":    {"spam", "标为垃圾"},
}

// moderator 生成并校验邮件里的评论审核链接。链接带 HMAC 签名，本身就是凭证，不需要登录。
type moderator struct {
	key []byte
}

// newModerator 用 secret（配置里的密码哈希）派生签名密钥；改密码会让旧链接失效。
// secret 为空时用随机密钥，重启后旧链接失效。
func newModerator(secret string) *moderator {
	if secret == "" {
		slog.Warn("no password in config, comment moderation links expire on restart")
		secret = rand.Text()
	}
	sum := sha256.Sum256([]byte("smoe-comment-moderation\x00" + secret))
	return &moderator{key: sum[:]}
}

func (m *moderator) sign(post string, id uint, action string, exp int64) string {
	mac := hmac.New(sha256.New, m.key)
	fmt.Fprintf(mac, "%s\x00%d\x00%s\x00%d", post, id, action, exp)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// url 返回某条评论的审核链接，base 是站点地址（不带尾部斜杠）。
func (m *moderator) url(base, post string, id uint, action string) string {
	exp := time.Now().Add(moderateTTL).Unix()
	q := url.Values{}
	q.Set("post", post)
	q.Set("id", strconv.FormatUint(uint64(id), 10))
	q.Set("action", action)
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", m.sign(post, id, action, exp))
	return base + "/comment/moderate?" + q.Encode()
}

func (m *moderator) verify(r *http.Request) (post string, id uint, action string, err error) {
	q := r.URL.Query()
	post, action = q.Get("post"), q.Get("action")
	n, err1 := strconv.ParseUint(q.Get("id"), 10, 32)
	exp, err2 := strconv.ParseInt(q.Get("exp"), 10, 64)
	if err1 != nil || err2 != nil {
		return "", 0, "", errors.New("链接不完整")
	}
	if !hmac.Equal([]byte(q.Get("sig")), []byte(m.sign(post, uint(n), action, exp))) {
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

// moderate 处理审核链接：GET 只显示确认页（邮件客户端会预取链接，不能直接生效），POST 才执行。
func (s *Smoe) moderate(w http.ResponseWriter, r *http.Request) {
	render := func(code int, v map[string]any) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(code)
		_ = moderatePage.Execute(w, v)
	}
	post, id, action, err := s.mod.verify(r)
	if err != nil {
		render(http.StatusForbidden, map[string]any{"Error": err.Error()})
		return
	}
	act := moderateActions[action]
	confirm := r.Method != http.MethodPost

	var cm blogdata.Comment
	if confirm {
		cm, err = s.data.Comment(post, id)
	} else {
		cm, err = s.data.SetCommentStatus(post, id, act.status)
	}
	if err != nil {
		render(http.StatusNotFound, map[string]any{"Error": "评论不存在：" + err.Error()})
		return
	}

	title, _ := s.site.Post(post)
	heading := "已" + act.verb
	if confirm {
		heading = act.verb + "这条评论？"
	}
	render(http.StatusOK, map[string]any{
		"C":         cm,
		"Verb":      act.verb,
		"PostTitle": title,
		"PostURL":   "/archives/" + post + "#" + strconv.FormatUint(uint64(id), 10),
		"Confirm":   confirm,
		"Heading":   heading,
	})
}
