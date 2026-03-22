package mymiddleware

import (
	"SMOE/moe/store"
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

func SetCommentNotification(c *echo.Context, notification store.CommentNotification) {
	c.Set(commentNotificationKey, notification)
}

func ConfigureCommentNotifier(cfg store.Config) {
	defaultCommentNotifier = newCommentNotifier(cfg)
}

func newCommentNotifier(cfg store.Config) *commentNotifier {
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

func NotifyComment(notification store.CommentNotification) error {
	return defaultCommentNotifier.NotifyComment(notification)
}

func (n *commentNotifier) NotifyComment(notification store.CommentNotification) error {
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

			notification, ok := c.Get(commentNotificationKey).(store.CommentNotification)
			if !ok {
				return nil
			}

			go func(notification store.CommentNotification) {
				if err := sendCommentNotification(notification); err != nil {
					slog.Error("send comment notification failed", "cid", notification.PostCID, "coid", notification.Comment.Coid, "err", err)
				}
			}(notification)
			return nil
		}
	}
}

func buildSubject(notification store.CommentNotification) string {
	title := notification.PostTitle
	if strings.TrimSpace(title) == "" {
		title = fmt.Sprintf("文章 %d", notification.PostCID)
	}
	if notification.Parent != nil {
		return fmt.Sprintf("你的评论收到了回复 | %s", title)
	}
	return fmt.Sprintf("文章有新评论 | %s", title)
}

func buildTextBody(notification store.CommentNotification) string {
	var b strings.Builder
	if notification.Parent != nil {
		b.WriteString("你收到了新的评论回复。\n\n")
	} else {
		b.WriteString("站点收到了新的评论。\n\n")
	}

	fmt.Fprintf(&b, "文章: %s\n", displayPostTitle(notification))
	fmt.Fprintf(&b, "文章 ID: %d\n", notification.PostCID)
	fmt.Fprintf(&b, "评论者: %s <%s>\n", notification.Comment.Author, notification.Comment.Mail)
	if notification.Comment.Url != nil && strings.TrimSpace(*notification.Comment.Url) != "" {
		fmt.Fprintf(&b, "站点: %s\n", *notification.Comment.Url)
	}
	if notification.Parent != nil {
		fmt.Fprintf(&b, "被回复评论者: %s <%s>\n", notification.Parent.Author, notification.Parent.Mail)
	}
	b.WriteString("\n评论内容:\n")
	b.WriteString(notification.Comment.Text)
	b.WriteString("\n")
	return b.String()
}

func buildHTMLBody(notification store.CommentNotification) string {
	var b strings.Builder
	b.WriteString("<h2>")
	if notification.Parent != nil {
		b.WriteString("你收到了新的评论回复")
	} else {
		b.WriteString("站点收到了新的评论")
	}
	b.WriteString("</h2>")
	fmt.Fprintf(&b, "<p><strong>文章:</strong> %s</p>", html.EscapeString(displayPostTitle(notification)))
	fmt.Fprintf(&b, "<p><strong>文章 ID:</strong> %d</p>", notification.PostCID)
	fmt.Fprintf(&b, "<p><strong>评论者:</strong> %s &lt;%s&gt;</p>", html.EscapeString(notification.Comment.Author), html.EscapeString(notification.Comment.Mail))
	if notification.Comment.Url != nil && strings.TrimSpace(*notification.Comment.Url) != "" {
		fmt.Fprintf(&b, "<p><strong>站点:</strong> %s</p>", html.EscapeString(*notification.Comment.Url))
	}
	if notification.Parent != nil {
		fmt.Fprintf(&b, "<p><strong>被回复评论者:</strong> %s &lt;%s&gt;</p>", html.EscapeString(notification.Parent.Author), html.EscapeString(notification.Parent.Mail))
	}
	fmt.Fprintf(&b, "<p><strong>评论内容:</strong></p><pre>%s</pre>", html.EscapeString(notification.Comment.Text))
	return b.String()
}

func displayPostTitle(notification store.CommentNotification) string {
	if strings.TrimSpace(notification.PostTitle) != "" {
		return notification.PostTitle
	}
	return fmt.Sprintf("文章 %d", notification.PostCID)
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
