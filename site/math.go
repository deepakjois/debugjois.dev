package main

import (
	"bytes"
	"io"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer"
	"github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/text"
	"github.com/yuin/goldmark/v2/util"
)

// The math extension supports the TeX math syntax of Obsidian:
//
//	Inline math: $e^{i\pi} = -1$
//
//	Display math:
//	$$
//	\frac{\rho}{1-\rho}
//	$$
//
// It renders the TeX source into elements with the class "math", which
// KaTeX renders in the browser (see templates/shell.html).

// InlineMath is TeX math within a paragraph, written as $...$ or, for
// display style, $$...$$.
type InlineMath struct {
	ast.BaseInline
	Display bool
	Value   text.SingleLineValue
}

// KindInlineMath is the NodeKind of InlineMath nodes.
var KindInlineMath = ast.NewNodeKind("InlineMath")

// Kind implements ast.Node.
func (n *InlineMath) Kind() ast.NodeKind { return KindInlineMath }

// Dump implements ast.Node.
func (n *InlineMath) Dump(_ []byte) *ast.NodeDump {
	return ast.NewNodeDump(n, map[string]any{"Display": n.Display, "Value": n.Value})
}

// MathBlock is display math written as a block that starts and ends with a
// line containing $$.
type MathBlock struct {
	ast.BaseBlock
	Value text.Lines
	// closed reports whether the closing $$ has been seen.
	closed bool
}

// KindMathBlock is the NodeKind of MathBlock nodes.
var KindMathBlock = ast.NewNodeKind("MathBlock")

// Kind implements ast.Node.
func (n *MathBlock) Kind() ast.NodeKind { return KindMathBlock }

// Dump implements ast.Node.
func (n *MathBlock) Dump(_ []byte) *ast.NodeDump {
	return ast.NewNodeDump(n, map[string]any{"Value": n.Value})
}

type inlineMathParser struct{}

func (p *inlineMathParser) Trigger() []byte { return []byte{'$'} }

// Parse parses $...$ and $$...$$ on a single line. Like Pandoc and
// Obsidian, the opening $ must not be followed by a space and the closing
// $ must not be preceded by a space or followed by a digit, so that
// amounts such as "$5 and $10" stay text.
func (p *inlineMathParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	if block.PrecedingCharacter() == '$' {
		return nil
	}
	line, seg := block.PeekLine()
	delim := 1
	if len(line) > 1 && line[1] == '$' {
		delim = 2
	}
	start := delim
	if start >= len(line) || util.IsSpace(line[start]) || line[start] == '$' {
		return nil
	}
	stop := -1
	for i := start; i < len(line); i++ {
		if line[i] == '\\' {
			i++
			continue
		}
		if line[i] == '$' {
			stop = i
			break
		}
	}
	if stop < 0 || util.IsSpace(line[stop-1]) || !bytes.HasPrefix(line[stop:], []byte("$$")[:delim]) {
		return nil
	}
	if after := stop + delim; delim == 1 && after < len(line) && (line[after] >= '0' && line[after] <= '9' || line[after] == '$') {
		return nil
	}
	n := &InlineMath{
		Display: delim == 2,
		Value:   text.NewSingleLineValueFromIndex(text.NewIndex(seg.Start+start, seg.Start+stop), text.IdentityDecoder),
	}
	n.Init(n)
	block.Advance(stop + delim)
	return n
}

type mathBlockParser struct{}

func (b *mathBlockParser) Trigger() []byte { return []byte{'$'} }

// Open accepts a line that is either just $$, which starts a block that
// ends with the next line ending in $$, or a complete $$...$$.
func (b *mathBlockParser) Open(_ ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, seg := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, parser.NoChildren
	}
	rest := util.TrimRightSpace(line[pos:])
	if !bytes.HasPrefix(rest, []byte("$$")) {
		return nil, parser.NoChildren
	}
	n := &MathBlock{}
	n.Init(n)
	if len(rest) > 2 {
		inner := rest[2:]
		if len(inner) < 3 || !bytes.HasSuffix(inner, []byte("$$")) || bytes.Contains(inner[:len(inner)-2], []byte("$$")) {
			return nil, parser.NoChildren
		}
		start := seg.Start + pos + 2
		n.Value.AppendSegment(text.NewSegment(start, start+len(inner)-2))
		n.closed = true
	}
	reader.AdvanceToEOL()
	return n, parser.NoChildren
}

func (b *mathBlockParser) Continue(node ast.Node, reader text.Reader, _ parser.Context) parser.State {
	n := node.(*MathBlock)
	if n.closed {
		return parser.Close
	}
	line, seg := reader.PeekLine()
	trimmed := util.TrimRightSpace(line)
	if bytes.HasSuffix(trimmed, []byte("$$")) {
		if len(trimmed) > 2 {
			n.Value.AppendSegment(seg.WithStop(seg.Start + len(trimmed) - 2))
		}
		reader.AdvanceToEOL()
		n.closed = true
		return parser.Close
	}
	n.Value.AppendSegment(seg)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (b *mathBlockParser) Close(ast.Node, text.Reader, parser.Context) {}

func (b *mathBlockParser) CanInterruptParagraph() bool { return true }

func (b *mathBlockParser) CanAcceptIndentedLine() bool { return false }

func newMathParser() parser.Extension { return mathParserExtension{} }

type mathParserExtension struct{}

func (mathParserExtension) ParserOptions(*parser.Config) []parser.Option {
	return []parser.Option{
		parser.WithBlockParsers(util.Prioritized[parser.BlockParser](&mathBlockParser{}, 750)),
		parser.WithInlineParsers(util.Prioritized[parser.InlineParser](&inlineMathParser{}, 600)),
	}
}

func newMathHTMLRenderer() html.Extension { return mathHTMLRendererExtension{} }

type mathHTMLRendererExtension struct{}

func (mathHTMLRendererExtension) RendererOptions(*html.Config) []html.Option {
	return []html.Option{
		html.WithNodeRenderers(map[ast.NodeKind]html.NodeRenderer{
			KindInlineMath: html.NodeRendererFunc(renderInlineMath),
			KindMathBlock:  html.NodeRendererFunc(renderMathBlock),
		}),
	}
}

func renderInlineMath(w io.Writer, source []byte, node ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*InlineMath)
	if n.Display {
		_, _ = io.WriteString(w, `<span class="math math-display">`)
	} else {
		_, _ = io.WriteString(w, `<span class="math math-inline">`)
	}
	_, _ = n.Value.WriteTo(html.ContextTextWriter(rc), source)
	_, _ = io.WriteString(w, `</span>`)
	return ast.WalkSkipChildren, nil
}

func renderMathBlock(w io.Writer, source []byte, node ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	_, _ = io.WriteString(w, `<div class="math math-display">`)
	_, _ = node.(*MathBlock).Value.WriteTo(html.ContextTextWriter(rc), source)
	_, _ = io.WriteString(w, "</div>\n")
	return ast.WalkSkipChildren, nil
}
