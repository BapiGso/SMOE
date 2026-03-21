package store

import (
	latex "github.com/aziis98/goldmark-latex"
	figure "github.com/mangoumbrella/goldmark-figure"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/mermaid"
)

var goldMark = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Linkify,
		figure.Figure,
		&frontmatter.Extender{},
		&mermaid.Extender{
			MermaidURL: "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js",
		},
		latex.NewLatex(
			latex.WithOutputInlineDelim(`\(`, `\)`),
			latex.WithOutputBlockDelim(`\[`, `\]`),
		),
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
		),
	),

	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithUnsafe(),
	),
)
