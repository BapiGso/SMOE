package moe

import (
	"cmp"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/BapiGso/SMOE/internal/moe/blogdata"

	"github.com/resend/resend-go/v3"
)

// submitComment 保存一条待审核评论，并发邮件提醒（带审核链接）。
func (s *Smoe) submitComment(w http.ResponseWriter, r *http.Request) {
	if !s.commentLimit.allow(clientIP(r)) {
		s.fail(w, http.StatusTooManyRequests)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	c, ok := parseComment(r)
	if !ok {
		s.fail(w, http.StatusBadRequest)
		return
	}
	cid := r.PathValue("cid")
	title, ok := s.site.Post(cid)
	if !ok {
		s.fail(w, http.StatusNotFound)
		return
	}
	c, parent, err := s.data.AddComment(cid, c)
	if errors.Is(err, blogdata.ErrNoComment) { // 回复的评论不存在或未通过
		s.fail(w, http.StatusBadRequest)
		return
	}
	if err != nil {
		slog.Error("save comment", "post", cid, "err", err)
		s.fail(w, http.StatusInternalServerError)
		return
	}
	if s.mail != nil {
		base := s.site.BaseURL()
		go s.mail.send(notification{
			PostTitle:  title,
			PostURL:    base + "/archives/" + cid,
			Comment:    c,
			Parent:     parent,
			ApproveURL: s.mod.url(base, cid, c.ID, "approve"),
			SpamURL:    s.mod.url(base, cid, c.ID, "spam"),
		})
	}
	// 200 空响应，前端据此提示“审核后显示”
}

// parseComment 读取并校验评论表单。长度按字符数算。
func parseComment(r *http.Request) (blogdata.Comment, bool) {
	c := blogdata.Comment{
		Author:  r.PostFormValue("author"),
		Mail:    r.PostFormValue("mail"),
		URL:     r.PostFormValue("url"),
		Content: r.PostFormValue("text"),
	}
	if v := r.PostFormValue("parent"); v != "" && v != "0" {
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return c, false
		}
		c.Parent = uint(n)
	}
	n := utf8.RuneCountInString
	addr, err := mail.ParseAddress(c.Mail)
	ok := c.Author != "" && n(c.Author) <= 50 &&
		err == nil && addr.Address == c.Mail && n(c.Mail) <= 50 &&
		c.Content != "" && n(c.Content) <= 1000 &&
		(c.URL == "" || n(c.URL) <= 200 && isHTTPURL(c.URL))
	return c, ok
}

// isHTTPURL 报告 s 是否是 http(s) 绝对地址。评论者站点会写进 href，不能放过 javascript: 之类。
func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// like 给文章点赞，同一访客对同一篇在冷却期内只算一次；计数在下一次刷新时写进 data/stats.json。
func (s *Smoe) like(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	if _, ok := s.site.Post(cid); !ok {
		s.fail(w, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !s.likeLimit.allow(clientIP(r) + "\x00" + cid) {
		_, _ = w.Write([]byte(`{"liked":false}`))
		return
	}
	s.data.Like(cid)
	_, _ = w.Write([]byte(`{"liked":true}`))
}

// mailer 用 Resend 给站长发新评论提醒。
type mailer struct {
	client *resend.Client
	to     string
}

// newMailer 在配置了 resendAPI 和 mailTo 时返回 mailer，否则返回 nil（不发邮件）。
func newMailer(cfg config) *mailer {
	to := strings.TrimSpace(cfg.MailTo)
	if cfg.ResendAPI == "" || to == "" {
		return nil
	}
	return &mailer{client: resend.NewClient(cfg.ResendAPI), to: to}
}

// notification 是一条新评论的提醒内容。
type notification struct {
	PostTitle  string
	PostURL    string
	Comment    blogdata.Comment
	Parent     *blogdata.Comment // 被回复的评论
	ApproveURL string            // 带签名的审核链接，点开后还需确认
	SpamURL    string
}

var mailBody = template.Must(template.New("mail").Parse(`<h2>{{ if .Parent }}你收到了新的评论回复{{ else }}站点收到了新的评论{{ end }}</h2>
<p><strong>文章:</strong> <a href="{{ .PostURL }}">{{ or .PostTitle "未命名文章" }}</a></p>
<p><strong>评论者:</strong> {{ .Comment.Author }} &lt;{{ .Comment.Mail }}&gt;</p>
{{- with .Comment.URL }}
<p><strong>站点:</strong> {{ . }}</p>
{{- end }}
{{- with .Parent }}
<p><strong>被回复评论者:</strong> {{ .Author }} &lt;{{ .Mail }}&gt;</p>
{{- end }}
<p><strong>评论内容:</strong></p><pre>{{ .Comment.Content }}</pre>
<p><a href="{{ .ApproveURL }}">通过</a> · <a href="{{ .SpamURL }}">标为垃圾</a></p>
`))

func (m *mailer) send(n notification) {
	var body strings.Builder
	err := mailBody.Execute(&body, n)
	if err == nil {
		subject := "文章有新评论 | "
		if n.Parent != nil {
			subject = "你的评论收到了回复 | "
		}
		_, err = m.client.Emails.Send(&resend.SendEmailRequest{
			From:    "onboarding@resend.dev",
			To:      []string{m.to},
			Subject: subject + cmp.Or(n.PostTitle, "未命名文章"),
			Html:    body.String(),
			ReplyTo: n.Comment.Mail,
		})
	}
	if err != nil {
		slog.Error("send comment notification failed", "post", n.PostURL, "comment", n.Comment.ID, "err", err)
	}
}
