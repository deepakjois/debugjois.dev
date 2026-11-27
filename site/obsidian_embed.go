package main

import (
	"io"
	"net/url"
	"strings"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer"
	"github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/text"
	"github.com/yuin/goldmark/v2/util"
)

// YouTube represents a YouTube video embedded with
// ![](https://www.youtube.com/watch?v=...) or ![](https://youtu.be/...).
// Its children are the image's alt text.
type YouTube struct {
	ast.BaseInline
	Video string
}

// KindYouTube is the NodeKind of YouTube nodes.
var KindYouTube = ast.NewNodeKind("YouTube")

// Kind implements ast.Node.
func (n *YouTube) Kind() ast.NodeKind { return KindYouTube }

// Dump implements ast.Node.
func (n *YouTube) Dump(_ []byte) *ast.NodeDump {
	return ast.NewNodeDump(n, map[string]any{"Video": n.Video})
}

// Twitter represents a Twitter/X post embedded with
// ![](https://x.com/.../status/...). Its children are the image's alt text.
type Twitter struct {
	ast.BaseInline
	TweetID string
}

// KindTwitter is the NodeKind of Twitter nodes.
var KindTwitter = ast.NewNodeKind("Twitter")

// Kind implements ast.Node.
func (n *Twitter) Kind() ast.NodeKind { return KindTwitter }

// Dump implements ast.Node.
func (n *Twitter) Dump(_ []byte) *ast.NodeDump {
	return ast.NewNodeDump(n, map[string]any{"TweetID": n.TweetID})
}

// embedTransformer replaces images that point to YouTube videos or tweets
// with embeds.
type embedTransformer struct{}

func (embedTransformer) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	var images []*ast.Image
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if img, ok := n.(*ast.Image); ok && entering {
			images = append(images, img)
		}
		return ast.WalkContinue, nil
	})

	for _, img := range images {
		u, err := url.Parse(img.Destination.Value(reader.Source()))
		if err != nil {
			continue
		}
		var embed ast.Node
		switch {
		case u.Host == "www.youtube.com" && u.Path == "/watch", u.Host == "youtu.be":
			v := u.Query().Get("v")
			if u.Host == "youtu.be" {
				v = strings.TrimPrefix(u.Path, "/")
			}
			if v != "" {
				yt := &YouTube{Video: v}
				yt.Init(yt)
				embed = yt
			}
		case u.Host == "twitter.com" || u.Host == "x.com":
			parts := strings.Split(u.Path, "/")
			if len(parts) >= 4 && parts[2] == "status" {
				tw := &Twitter{TweetID: parts[3]}
				tw.Init(tw)
				embed = tw
			}
		}
		if embed != nil {
			for img.HasChildren() {
				embed.AppendChild(img.FirstChild())
			}
			img.Parent().ReplaceChild(img, embed)
		}
	}
}

func newObsidianEmbedParser() parser.Extension { return obsidianEmbedParserExtension{} }

type obsidianEmbedParserExtension struct{}

func (obsidianEmbedParserExtension) ParserOptions(*parser.Config) []parser.Option {
	return []parser.Option{
		parser.WithASTTransformers(util.Prioritized[parser.ASTTransformer](embedTransformer{}, 500)),
	}
}

func newObsidianEmbedHTMLRenderer() html.Extension { return obsidianEmbedHTMLRendererExtension{} }

type obsidianEmbedHTMLRendererExtension struct{}

func (obsidianEmbedHTMLRendererExtension) RendererOptions(*html.Config) []html.Option {
	return []html.Option{
		html.WithNodeRenderers(map[ast.NodeKind]html.NodeRenderer{
			KindYouTube: html.NodeRendererFunc(renderYouTube),
			KindTwitter: html.NodeRendererFunc(renderTwitter),
		}),
	}
}

func renderYouTube(w io.Writer, _ []byte, node ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
	if entering {
		return ast.WalkContinue, nil
	}
	_, _ = io.WriteString(w, `<iframe width="560" height="315" class="youtube-video" src="https://www.youtube.com/embed/`)
	_, _ = html.ContextTextWriter(rc).WriteString(node.(*YouTube).Video)
	_, _ = io.WriteString(w, `" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>`)
	return ast.WalkContinue, nil
}

func renderTwitter(w io.Writer, _ []byte, node ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
	if entering {
		return ast.WalkContinue, nil
	}
	_, _ = io.WriteString(w, `<blockquote class="twitter-tweet" data-theme="dark"><a href="https://twitter.com/x/status/`)
	_, _ = html.ContextTextWriter(rc).WriteString(node.(*Twitter).TweetID)
	_, _ = io.WriteString(w, `"></a></blockquote><script async src="https://platform.twitter.com/widgets.js" charset="utf-8"></script>`)
	return ast.WalkContinue, nil
}
