package moe

import (
	"SMOE/moe/blogdata"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

func (s *Smoe) Listen() {
	port := s.cfg.Port
	if port == "" {
		port = "95"
	}
	fmt.Printf(banner, "=> http :"+port)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           s.e,
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
	defer func() {
		jobs.Wait()
		// 关停期间还在处理的请求也可能记了数
		if err := s.data.Flush(); err != nil {
			slog.Error("blogdata: flush on exit failed", "err", err)
		}
		if err := s.site.Close(); err != nil {
			slog.Warn("engine close", "err", err)
		}
	}()

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
}
