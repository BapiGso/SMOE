package moe

import (
	"SMOE/assets"
	"SMOE/moe/store"
	"SMOE/moe/tools"
	"embed"
	"log"

	"github.com/labstack/echo/v5"
)

type Smoe struct {
	cfg     store.Config
	themeFS *embed.FS    //主题所在文件夹
	e       *echo.Echo   //后台框架
	mail    *tools.Email //邮件提醒
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
	cfg, err := store.ReadConfig()
	if err != nil {
		log.Fatal("读取 usr/config.yaml 失败: ", err)
	}
	s := &Smoe{cfg: cfg}
	s.themeFS = &assets.Assets
	s.e = echo.New()
	return s
}
