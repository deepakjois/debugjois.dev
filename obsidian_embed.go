package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// YouTube struct represents a YouTube Video embed of the Markdown text.
type YouTube struct {
	ast.Image
	Video string
}

// KindYouTube is a NodeKind of the YouTube node.
var KindYouTube = ast.NewNodeKind("YouTube")

// Kind implements Node.Kind.
func (n *YouTube) Kind() ast.NodeKind {
	return KindYouTube
}

// NewYouTube returns a new YouTube node.
func NewYouTube(img *ast.Image, v string) *YouTube {
	c := &YouTube{
		Image: *img,
		Video: v,
	}
	c.Destination = img.Destination
	c.Title = img.Title

	return c
}

// Twitter struct represents a Twitter/X embed of the Markdown text.
type Twitter struct {
	ast.Image
	TweetID string
}

// KindTwitter is a NodeKind of the Twitter node.
var KindTwitter = ast.NewNodeKind("Twitter")

// Kind implements Node.Kind.
func (n *Twitter) Kind() ast.NodeKind {
	return KindTwitter
}

// NewTwitter returns a new Twitter node.
func NewTwitter(img *ast.Image, tweetID string) *Twitter {
	c := &Twitter{
		Image:   *img,
		TweetID: tweetID,
	}
	c.Destination = img.Destination
	c.Title = img.Title

	return c
}

type astTransformer struct{}

var defaultASTTransformer = &astTransformer{}

func (a *astTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	replaceImages := func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() != ast.KindImage {
			return ast.WalkContinue, nil
		}

		img, ok := n.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}

		u, err := url.Parse(string(img.Destination))
		if err != nil {
			msg := ast.NewString([]byte(fmt.Sprintf("<!-- %s -->", err)))
			msg.SetCode(true)
			n.Parent().InsertAfter(n.Parent(), n, msg)
			return ast.WalkContinue, nil
		}

		switch {
		case u.Host == "www.youtube.com" && u.Path == "/watch":
			v := u.Query().Get("v")
			if v != "" {
				yt := NewYouTube(img, v)
				n.Parent().ReplaceChild(n.Parent(), n, yt)
			}
		case u.Host == "twitter.com" || u.Host == "x.com":
			parts := strings.Split(u.Path, "/")
			if len(parts) >= 4 && parts[2] == "status" {
				tw := NewTwitter(img, parts[3])
				n.Parent().ReplaceChild(n.Parent(), n, tw)
			}
		}

		return ast.WalkContinue, nil
	}

	ast.Walk(node, replaceImages)
}

// HTMLRenderer struct is a renderer.NodeRenderer implementation for the extension.
type HTMLRenderer struct{}

// NewHTMLRenderer builds a new HTMLRenderer with given options and returns it.
func NewHTMLRenderer() renderer.NodeRenderer {
	r := &HTMLRenderer{}
	return r
}

// RegisterFuncs implements NodeRenderer.RegisterFuncs.
func (r *HTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindYouTube, r.renderYouTubeVideo)
	reg.Register(KindTwitter, r.renderTwitterEmbed)
}

func (r *HTMLRenderer) renderYouTubeVideo(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		return ast.WalkContinue, nil
	}

	yt := node.(*YouTube)
	w.Write([]byte(`<iframe width="560" height="315" class="youtube-video" src="https://www.youtube.com/embed/` + yt.Video + `" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>`))
	return ast.WalkContinue, nil
}

func (r *HTMLRenderer) renderTwitterEmbed(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		return ast.WalkContinue, nil
	}

	tw := node.(*Twitter)
	w.Write([]byte(`<blockquote class="twitter-tweet" data-theme="dark"><a href="https://twitter.com/x/status/` + tw.TweetID + `"></a></blockquote><script async src="https://platform.twitter.com/widgets.js" charset="utf-8"></script>`))
	return ast.WalkContinue, nil
}

type ObsidianEmbedExtender struct{}

func (e *ObsidianEmbedExtender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(defaultASTTransformer, 500),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewHTMLRenderer(), 500),
		),
	)
}
