package assets

import (
	"embed"
)

var (
	// theme 是 Hugo 主题（含 _partials、_markup 等下划线目录，所以要 all:）
	//go:embed all:theme
	Assets embed.FS
)
