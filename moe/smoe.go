package moe

import (
	"SMOE/assets"
	"SMOE/moe/blogdata"
	"SMOE/moe/config"
	"SMOE/moe/engine"
	"SMOE/moe/handler"
	"SMOE/moe/migrate"
	"SMOE/moe/mymiddleware"
	"io/fs"
	"log"
	"log/slog"
	"path/filepath"

	"github.com/labstack/echo/v5"
)

type Smoe struct {
	cfg  config.Config
	e    *echo.Echo         //HTTP 框架
	site *engine.Engine     //Hugo 构建出的前台
	data *blogdata.Store    //浏览量、点赞、评论（usr/data）
	mod  *handler.Moderator //邮件里的评论审核链接
}

const (
	banner = `
 ______     __    __     ______     ______
/\  ___\   /\ \-./  \   /\  __ \   /\  ___\
\ \___  \  \ \ \-./\ \  \ \ \/\ \  \ \  __\
 \/\_____\  \ \_\ \ \_\  \ \_____\  \ \_____\
  \/_____/   \/_/  \/_/   \/_____/   \/_____/

____________________________________O/_______
                                    O\
%s
`
)

func New() *Smoe {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal("读取 usr/config.yaml 失败: ", err)
	}
	s := &Smoe{cfg: cfg, e: echo.New()}
	mymiddleware.ConfigureCommentNotifier(cfg)

	// 旧版 usr/posts、usr/pages → usr/content、usr/data，只在两者都不存在时做一次
	if done, err := migrate.Run("usr"); err != nil {
		log.Fatal("迁移旧数据失败（旧文件未改动）: ", err)
	} else if done {
		slog.Info("已迁移到 usr/content 和 usr/data，旧的 usr/posts、usr/pages 保留未动")
	}

	theme, err := fs.Sub(assets.Assets, "theme")
	if err != nil {
		log.Fatal(err)
	}
	if s.site, err = engine.New("usr", theme); err != nil {
		log.Fatal("初始化 Hugo 失败: ", err)
	}
	// 写进 data/ 的东西会出现在页面上，写完触发重建
	if s.data, err = blogdata.Open(filepath.Join("usr", "data"), s.site.Trigger); err != nil {
		log.Fatal("读取 usr/data 失败: ", err)
	}
	s.mod = handler.NewModerator(cfg.Password)
	// 首次构建失败不退出：前台返回 503，WebDAV 仍可用来修文件，修好后自动重建
	if err := s.site.Build(); err != nil {
		slog.Error("hugo build failed, fix usr/content and it will rebuild", "err", err)
	}
	return s
}
