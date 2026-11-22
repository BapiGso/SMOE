package moe

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/BapiGso/SMOE/internal/moe/blogdata"
	"github.com/BapiGso/SMOE/internal/moe/engine"
	"github.com/BapiGso/SMOE/internal/moe/migrate"
)

type Smoe struct {
	cfg  config
	site *engine.Engine  // Hugo 构建出的前台
	data *blogdata.Store // 浏览量、点赞、评论（usr/data）
	mod  *moderator      // 邮件里的评论审核链接
	mail *mailer         // 新评论提醒，没配置时为 nil

	// 同一访客的评论间隔、同一访客对同一篇的点赞和浏览去重
	commentLimit, likeLimit, viewSeen *cooldown
}

const banner = `
 ______     __    __     ______     ______
/\  ___\   /\ \-./  \   /\  __ \   /\  ___\
\ \___  \  \ \ \-./\ \  \ \ \/\ \  \ \  __\
 \/\_____\  \ \_\ \ \_\  \ \_____\  \ \_____\
  \/_____/   \/_/  \/_/   \/_____/   \/_____/

____________________________________O/_______
                                    O\
%s
`

// New 读取配置、迁移旧数据并完成首次构建。theme 是 site.toml 和 smoe/ 主题目录所在的文件系统。
func New(theme fs.FS) *Smoe {
	cfg, err := readConfig()
	if err != nil {
		log.Fatal("读取 usr/config.yaml 失败: ", err)
	}
	s := &Smoe{
		cfg:          cfg,
		mod:          newModerator(cfg.Password),
		mail:         newMailer(cfg),
		commentLimit: &cooldown{window: time.Minute},
		likeLimit:    &cooldown{window: 6 * time.Hour},
		viewSeen:     &cooldown{window: 6 * time.Hour},
	}

	// 旧版 usr/posts、usr/pages → usr/content、usr/data，只在两者都不存在时做一次
	if done, err := migrate.Run("usr"); err != nil {
		log.Fatal("迁移旧数据失败（旧文件未改动）: ", err)
	} else if done {
		slog.Info("已迁移到 usr/content 和 usr/data，旧的 usr/posts、usr/pages 保留未动")
	}

	if s.site, err = engine.New("usr", theme); err != nil {
		log.Fatal("初始化 Hugo 失败: ", err)
	}
	// 写进 data/ 的东西会出现在页面上，写完触发重建
	if s.data, err = blogdata.Open(filepath.Join("usr", "data"), s.site.Trigger); err != nil {
		log.Fatal("读取 usr/data 失败: ", err)
	}
	// 首次构建失败不退出：前台返回 503，WebDAV 仍可用来修文件，修好后自动重建
	if err := s.site.Build(); err != nil {
		slog.Error("hugo build failed, fix usr/content and it will rebuild", "err", err)
	}
	return s
}

// Listen 提供 HTTP 服务并运行后台任务，收到 SIGINT/SIGTERM 后排空连接、写盘再返回。
func (s *Smoe) Listen() {
	port := cmp.Or(s.cfg.Port, "95")
	fmt.Printf(banner, "=> http :"+port)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           s.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 后台任务：监听 usr/ 改动重建站点、定期写入浏览量和点赞、定期刷新番剧数据
	var jobs sync.WaitGroup
	jobs.Go(func() { s.site.Run(ctx) })
	jobs.Go(func() { s.data.Run(ctx, 10*time.Minute) })
	jobs.Go(func() { blogdata.RunBangumi(ctx, filepath.Join("usr", "data"), s.cfg.BangumiUserID, s.site.Trigger) })

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server stopped", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, draining connections")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}

	jobs.Wait()
	// 连接排空之后再写，关停期间处理的请求也算数
	if err := s.data.Flush(); err != nil {
		slog.Error("blogdata: flush on exit failed", "err", err)
	}
	if err := s.site.Close(); err != nil {
		slog.Warn("engine close", "err", err)
	}
}
