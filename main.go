package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/BapiGso/SMOE/internal/moe"
)

// Hugo 主题（含 _partials、_markup 等下划线目录，所以要 all:）
//
//go:embed all:assets/theme
var assets embed.FS

func main() {
	theme, err := fs.Sub(assets, "assets/theme")
	if err != nil {
		log.Fatal(err)
	}
	moe.New(theme).Listen()
}
