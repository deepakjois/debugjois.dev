package main

import (
	"bytes"
	"io"
	"path"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer"
	"github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/text"
	"github.com/yuin/goldmark/v2/util"
)

// ObsidianImage represents an Obsidian-style embedded image: ![[file.png]]
type ObsidianImage struct {
	ast.BaseInline
	Filename string
}

// KindObsidianImage is the NodeKind of ObsidianImage nodes.
var KindObsidianImage = ast.NewNodeKind("ObsidianImage")

// Kind implements ast.Node.
func (n *ObsidianImage) Kind() ast.NodeKind { return KindObsidianImage }

// Dump implements ast.Node.
func (n *ObsidianImage) Dump(_ []byte) *ast.NodeDump {
	return ast.NewNodeDump(n, map[string]any{"Filename": n.Filename})
}

// obsidianImageParser parses Obsidian-style image embeds.
type obsidianImageParser struct {
	imagePath string
}

func (s *obsidianImageParser) Trigger() []byte { return []byte{'!'} }

func (s *obsidianImageParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if !bytes.HasPrefix(line, []byte("![[")) {
		return nil
	}
	end := bytes.Index(line[3:], []byte("]]"))
	if end <= 0 {
		return nil
	}
	n := &ObsidianImage{Filename: path.Join(s.imagePath, string(line[3:3+end]))}
	n.Init(n)
	block.Advance(3 + end + 2)
	return n
}

func newObsidianImageParser(imagePath string) parser.Extension {
	return obsidianImageParserExtension{imagePath: imagePath}
}

type obsidianImageParserExtension struct {
	imagePath string
}

func (e obsidianImageParserExtension) ParserOptions(*parser.Config) []parser.Option {
	return []parser.Option{
		parser.WithInlineParsers(
			util.Prioritized[parser.InlineParser](&obsidianImageParser{imagePath: e.imagePath}, 100),
		),
	}
}

func newObsidianImageHTMLRenderer() html.Extension { return obsidianImageHTMLRendererExtension{} }

type obsidianImageHTMLRendererExtension struct{}

func (obsidianImageHTMLRendererExtension) RendererOptions(*html.Config) []html.Option {
	return []html.Option{
		html.WithNodeRenderer(KindObsidianImage, html.NodeRendererFunc(renderObsidianImage)),
	}
}

func renderObsidianImage(w io.Writer, _ []byte, node ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ObsidianImage)
	tw := html.ContextTextWriter(rc)
	_, _ = io.WriteString(w, `<img src="`)
	_, _ = tw.WriteString(n.Filename)
	_, _ = io.WriteString(w, `" alt="`)
	_, _ = tw.WriteString(n.Filename)
	_, _ = io.WriteString(w, `">`)
	return ast.WalkSkipChildren, nil
}
