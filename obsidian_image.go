package main

import (
	"fmt"
	"path"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// ObsidianImage represents an Obsidian-style embedded image
type ObsidianImage struct {
	ast.BaseInline
	Filename string
}

// Kind returns the kind of node
func (n *ObsidianImage) Kind() ast.NodeKind {
	return ast.KindImage
}

// Dump dumps the node to a string
func (n *ObsidianImage) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// ObsidianImageParser parses Obsidian-style image embeds
type ObsidianImageParser struct {
	ImagePath string
}

// NewObsidianImageParser returns a new ObsidianImageParser
func NewObsidianImageParser(imagePath string) parser.InlineParser {
	return &ObsidianImageParser{ImagePath: imagePath}
}

// Trigger returns the characters that trigger this parser
func (s *ObsidianImageParser) Trigger() []byte {
	return []byte{'!'}
}

// Parse parses the Obsidian-style image embed
func (s *ObsidianImageParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) < 5 || line[1] != '[' || line[2] != '[' {
		return nil
	}

	block.Advance(3) // Skip "![["

	closeBracketPos := -1
	for i := 3; i < len(line)-1; i++ {
		if line[i] == ']' && line[i+1] == ']' {
			closeBracketPos = i
			break
		}
	}

	if closeBracketPos == -1 {
		return nil // No closing brackets found
	}

	filename := string(line[3:closeBracketPos])
	node := &ObsidianImage{Filename: path.Join(s.ImagePath, filename)}
	block.Advance(closeBracketPos - 3 + 2) // +2 for "]]"
	return node
}

// ObsidianImageRenderer renders ObsidianImage nodes
type ObsidianImageRenderer struct{}

// NewObsidianImageRenderer returns a new ObsidianImageRenderer
func NewObsidianImageRenderer() renderer.NodeRenderer {
	return &ObsidianImageRenderer{}
}

// RegisterFuncs registers the render functions
func (r *ObsidianImageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderObsidianImage)
}

func (r *ObsidianImageRenderer) renderObsidianImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n, ok := node.(*ObsidianImage)
	if !ok { // Regular image
		return ast.WalkContinue, nil
	}

	if _, err := w.WriteString(fmt.Sprintf(`<img src="%s" alt="%s">`, n.Filename, n.Filename)); err != nil {
		return ast.WalkSkipChildren, err
	}

	return ast.WalkSkipChildren, nil
}

// ObsidianImageExtender is a custom goldmark extension for Obsidian image embeds
type ObsidianImageExtender struct {
	ImagePath string
}

func (e *ObsidianImageExtender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(NewObsidianImageParser(e.ImagePath), 100),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewObsidianImageRenderer(), 100),
		),
	)
}
