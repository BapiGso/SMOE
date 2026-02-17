package moe

import (
	"SMOE/assets"
	"SMOE/moe/tools"
	"embed"
	"github.com/labstack/echo/v5"
)

type Smoe struct {
	param *bindFlag //命令行参数
	//Db      *sqlx.DB  //数据库
	themeFS *embed.FS //主题所在文件夹
	//mdParse *goldmark.Markdown //markdown->html解析器
	e    *echo.Echo   //后台框架
	mail *tools.Email //邮件提醒
	//异地多活
	//图片压缩webp
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

func New() (s *Smoe) {
	s = &Smoe{}
	s.themeFS = &assets.Assets
	s.e = echo.New()
	return s
}
