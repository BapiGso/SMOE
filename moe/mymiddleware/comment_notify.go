package mymiddleware

import (
	"SMOE/moe/blogdata"
	"SMOE/moe/config"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/resend/resend-go/v3"
)

const commentNotificationKey = "comment_notification"
const defaultCommentFrom = "onboarding@resend.dev"

type commentNotifier struct {
	client  *resend.Client
	enabled bool
	from    string
	to      string
	cc      string
}

var defaultCommentNotifier = &commentNotifier{}
var sendCommentNotification = NotifyComment

// CommentNotification 是一条新评论的提醒内容。
type CommentNotification struct {
	PostTitle  string
	PostURL    string
	Comment    blogdata.Comment
	Parent     *blogdata.Comment // 被回复的评论
	ApproveURL string            // 带签名的审核链接，点开后还需确认
	SpamURL    string
}

func SetCommentNotification(c *echo.Context, notification CommentNotification) {
	c.Set(commentNotificationKey, notification)
}

func ConfigureCommentNotifier(cfg config.Config) {
	defaultCommentNotifier = newCommentNotifier(cfg)
}

func newCommentNotifier(cfg config.Config) *commentNotifier {
	if cfg.ResendAPI == "" || cfg.MailTo == "" {
		return &commentNotifier{}
	}
	return &commentNotifier{
		client:  resend.NewClient(cfg.ResendAPI),
		enabled: true,
		from:    defaultCommentFrom,
		to:      strings.TrimSpace(cfg.MailTo),
		cc:      strings.TrimSpace(cfg.MailCC),
	}
}

func NotifyComment(notification CommentNotification) error {
	return defaultCommentNotifier.NotifyComment(notification)
}

func (n *commentNotifier) NotifyComment(notification CommentNotification) error {
	if n == nil || !n.enabled || n.client == nil {
		return nil
	}

	to := uniqueEmails([]string{n.to})
	if len(to) == 0 {
		return nil
	}
	cc := uniqueEmails([]string{n.cc}, to...)

	req := &resend.SendEmailRequest{
		From:    n.from,
		To:      to,
		Cc:      cc,
		Subject: buildSubject(notification),
		Text:    buildTextBody(notification),
		Html:    buildHTMLBody(notification),
		ReplyTo: strings.TrimSpace(notification.Comment.Mail),
	}
	_, err := n.client.Emails.Send(req)
	return err
}

func CommentNotify() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if err := next(c); err != nil {
				return err
			}

			notification, ok := c.Get(commentNotificationKey).(CommentNotification)
			if !ok {
				return nil
			}

			go func(notification CommentNotification) {
				if err := sendCommentNotification(notification); err != nil {
					slog.Error("send comment notification failed", "post", notification.PostURL, "comment", notification.Comment.ID, "err", err)
				}
			}(notification)
			return nil
		}
	}
}

func buildSubject(notification CommentNotification) string {
	if notification.Parent != nil {
		return fmt.Sprintf("你的评论收到了回复 | %s", displayPostTitle(notification))
	}
	return fmt.Sprintf("文章有新评论 | %s", displayPostTitle(notification))
}

func buildTextBody(notification CommentNotification) string {
	var b strings.Builder
	if notification.Parent != nil {
		b.WriteString("你收到了新的评论回复。\n\n")
	} else {
		b.WriteString("站点收到了新的评论。\n\n")
	}

	fmt.Fprintf(&b, "文章: %s\n", displayPostTitle(notification))
	if notification.PostURL != "" {
		fmt.Fprintf(&b, "地址: %s\n", notification.PostURL)
	}
	fmt.Fprintf(&b, "评论者: %s <%s>\n", notification.Comment.Author, notification.Comment.Mail)
	if strings.TrimSpace(notification.Comment.URL) != "" {
		fmt.Fprintf(&b, "站点: %s\n", notification.Comment.URL)
	}
	if notification.Parent != nil {
		fmt.Fprintf(&b, "被回复评论者: %s <%s>\n", notification.Parent.Author, notification.Parent.Mail)
	}
	b.WriteString("\n评论内容:\n")
	b.WriteString(notification.Comment.Content)
	b.WriteString("\n")
	if notification.ApproveURL != "" {
		fmt.Fprintf(&b, "\n通过: %s\n标为垃圾: %s\n", notification.ApproveURL, notification.SpamURL)
	}
	return b.String()
}

func buildHTMLBody(notification CommentNotification) string {
	var b strings.Builder
	b.WriteString("<h2>")
	if notification.Parent != nil {
		b.WriteString("你收到了新的评论回复")
	} else {
		b.WriteString("站点收到了新的评论")
	}
	b.WriteString("</h2>")
	if notification.PostURL != "" {
		fmt.Fprintf(&b, `<p><strong>文章:</strong> <a href="%s">%s</a></p>`, html.EscapeString(notification.PostURL), html.EscapeString(displayPostTitle(notification)))
	} else {
		fmt.Fprintf(&b, "<p><strong>文章:</strong> %s</p>", html.EscapeString(displayPostTitle(notification)))
	}
	fmt.Fprintf(&b, "<p><strong>评论者:</strong> %s &lt;%s&gt;</p>", html.EscapeString(notification.Comment.Author), html.EscapeString(notification.Comment.Mail))
	if strings.TrimSpace(notification.Comment.URL) != "" {
		fmt.Fprintf(&b, "<p><strong>站点:</strong> %s</p>", html.EscapeString(notification.Comment.URL))
	}
	if notification.Parent != nil {
		fmt.Fprintf(&b, "<p><strong>被回复评论者:</strong> %s &lt;%s&gt;</p>", html.EscapeString(notification.Parent.Author), html.EscapeString(notification.Parent.Mail))
	}
	fmt.Fprintf(&b, "<p><strong>评论内容:</strong></p><pre>%s</pre>", html.EscapeString(notification.Comment.Content))
	if notification.ApproveURL != "" {
		fmt.Fprintf(&b, `<p><a href="%s">通过</a> · <a href="%s">标为垃圾</a></p>`, html.EscapeString(notification.ApproveURL), html.EscapeString(notification.SpamURL))
	}
	return b.String()
}

func displayPostTitle(notification CommentNotification) string {
	if strings.TrimSpace(notification.PostTitle) != "" {
		return notification.PostTitle
	}
	return "未命名文章"
}

func uniqueEmails(emails []string, exclude ...string) []string {
	seen := make(map[string]struct{}, len(emails)+len(exclude))
	for _, email := range exclude {
		key := strings.ToLower(strings.TrimSpace(email))
		if key != "" {
			seen[key] = struct{}{}
		}
	}

	out := make([]string, 0, len(emails))
	for _, email := range emails {
		trimmed := strings.TrimSpace(email)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
