package mymiddleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/BapiGso/SMOE/internal/moe/blogdata"
	"github.com/BapiGso/SMOE/internal/moe/config"

	"github.com/labstack/echo/v5"
)

func TestCommentNotify(t *testing.T) {
	orig := sendCommentNotification
	defer func() { sendCommentNotification = orig }()

	notified := make(chan CommentNotification, 1)
	sendCommentNotification = func(notification CommentNotification) error {
		notified <- notification
		return nil
	}

	e := echo.New()
	e.POST("/archives/:cid/comment", func(c *echo.Context) error {
		SetCommentNotification(c, CommentNotification{
			PostTitle: "中间件测试文章",
			PostURL:   "https://example.com/archives/42",
			Comment: blogdata.Comment{
				ID:      7,
				Author:  "tester",
				Mail:    "tester@example.com",
				Content: "middleware should trigger notification",
			},
		})
		return c.NoContent(http.StatusNoContent)
	}, CommentNotify())

	req := httptest.NewRequest(http.MethodPost, "/archives/42/comment", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	select {
	case notification := <-notified:
		if notification.PostURL != "https://example.com/archives/42" {
			t.Fatalf("expected post url, got %q", notification.PostURL)
		}
		if notification.Comment.ID != 7 {
			t.Fatalf("expected comment id 7, got %d", notification.Comment.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected comment notification to be sent")
	}
}

func TestSendCommentIntegration(t *testing.T) {
	if os.Getenv("SMOE_SEND_TEST_MAIL") != "1" {
		t.Skip("set SMOE_SEND_TEST_MAIL=1 to send a real test email")
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory failed: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("change working directory failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	cfg, err := config.Read()
	if err != nil {
		t.Fatalf("read config failed: %v", err)
	}
	if cfg.ResendAPI == "" || cfg.MailTo == "" {
		t.Skip("resendAPI or mailTo is empty")
	}

	notifier := newCommentNotifier(cfg)
	if !notifier.enabled {
		t.Skip("comment notifier is disabled")
	}

	notification := CommentNotification{
		PostTitle: "SMOE 评论通知测试",
		Comment: blogdata.Comment{
			ID:      uint(time.Now().Unix()),
			Author:  "Codex Test",
			Mail:    cfg.MailTo,
			Content: "这是一封通过 go test 发送的评论提醒测试邮件。",
		},
	}

	if err := notifier.NotifyComment(notification); err != nil {
		t.Fatalf("send test email failed: %v", err)
	}
}
