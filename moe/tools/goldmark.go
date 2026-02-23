package tools

import (
	latex "github.com/aziis98/goldmark-latex"
	mathjax "github.com/litao91/goldmark-mathjax"
	figure "github.com/mangoumbrella/goldmark-figure"
	treeblood "github.com/wyatt915/goldmark-treeblood"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/mermaid"
)

var GoldMark = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Linkify,
		figure.Figure,
		&frontmatter.Extender{},
		mathjax.MathJax,
		&mermaid.Extender{
			MermaidURL: "/assets/blog/js/mermaid.js", //自定义js静态资源路径
		},
		latex.NewLatex(),
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
		),
		treeblood.MathML(),
	),

	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithUnsafe(),
	),
)
