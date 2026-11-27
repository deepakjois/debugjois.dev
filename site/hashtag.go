package main

import (
	"bytes"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer"
	"github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/text"
	"github.com/yuin/goldmark/v2/util"
)

// Hashtag is an Obsidian-style #tag. Its child is the text of the tag,
// including the leading '#'.
type Hashtag struct {
	ast.BaseInline
}

// KindHashtag is the NodeKind of Hashtag nodes.
var KindHashtag = ast.NewNodeKind("Hashtag")

// Kind implements ast.Node.
func (n *Hashtag) Kind() ast.NodeKind { return KindHashtag }

// Dump implements ast.Node.
func (n *Hashtag) Dump(_ []byte) *ast.NodeDump { return ast.NewNodeDump(n, nil) }

type hashtagParser struct{}

func (p *hashtagParser) Trigger() []byte { return []byte{'#'} }

// Parse parses a hashtag following the rules of Obsidian
// (https://help.obsidian.md/tags): letters, digits, emoji and the symbols
// "_", "-" and "/", with at least one character that is not a digit.
func (p *hashtagParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, seg := block.PeekLine()
	tag := line[1:]
	end := 0
	for end < len(tag) {
		r, size := utf8.DecodeRune(tag[end:])
		if !isHashtagRune(r) {
			break
		}
		end += size
	}
	if bytes.IndexFunc(tag[:end], func(r rune) bool { return !unicode.IsDigit(r) }) < 0 {
		return nil
	}
	n := &Hashtag{}
	n.Init(n)
	n.AppendChild(ast.NewText(text.NewSingleLineValueFromIndex(
		text.NewIndex(seg.Start, seg.Start+1+end), block.Decoder(),
	)))
	block.Advance(1 + end)
	return n
}

// isHashtagRune reports whether r may appear in a hashtag. Marks are
// allowed so that combining accents and emoji modifiers (variation
// selectors, keycaps) stay in the tag. Emoji are approximated by the emoji
// blocks, the tag characters used by flag sequences, the zero width joiner
// and the emoji outside those blocks.
func isHashtagRune(r rune) bool {
	switch {
	case unicode.IsLetter(r), unicode.IsDigit(r), unicode.Is(unicode.Mark, r), r == '_', r == '-', r == '/':
		return true
	case r >= 0x2600 && r <= 0x27BF, r >= 0x1F000 && r <= 0x1FAFF, r >= 0xE0020 && r <= 0xE007F, r == '\u200d':
		return true
	}
	return strings.ContainsRune("©®‼⁉™↔↕↖↗↘↙↩↪⌚⌛⌨⏏⏩⏪⏫⏬⏭⏮⏯⏰⏱⏲⏳⏸⏹⏺Ⓜ▪▫▶◀◻◼◽◾⤴⤵⬅⬆⬇⬛⬜⭐⭕〰〽㊗㊙", r)
}

func newHashtagParser() parser.Extension { return hashtagParserExtension{} }

type hashtagParserExtension struct{}

func (hashtagParserExtension) ParserOptions(*parser.Config) []parser.Option {
	return []parser.Option{
		parser.WithInlineParsers(util.Prioritized[parser.InlineParser](&hashtagParser{}, 999)),
	}
}

func newHashtagHTMLRenderer() html.Extension { return hashtagHTMLRendererExtension{} }

type hashtagHTMLRendererExtension struct{}

func (hashtagHTMLRendererExtension) RendererOptions(*html.Config) []html.Option {
	return []html.Option{
		html.WithNodeRenderer(KindHashtag, html.NodeRendererFunc(renderHashtag)),
	}
}

func renderHashtag(w io.Writer, _ []byte, _ ast.Node, entering bool, _ renderer.Context) (ast.WalkStatus, error) {
	if entering {
		_, _ = io.WriteString(w, `<span class="hashtag">`)
	} else {
		_, _ = io.WriteString(w, `</span>`)
	}
	return ast.WalkContinue, nil
}
