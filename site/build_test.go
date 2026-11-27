package main

import (
	"bytes"
	"testing"
)

func TestMarkdownConvert(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "inline math",
			input: `Where $\rho$ is the rate.`,
			want:  `<p>Where <span class="math math-inline">\rho</span> is the rate.</p>` + "\n",
		},
		{
			name:  "dollar amounts are not math",
			input: `It cost $5 and $10, or $20,000 for \$3.`,
			want:  `<p>It cost $5 and $10, or $20,000 for $3.</p>` + "\n",
		},
		{
			name:  "closing dollar followed by digit is not math",
			input: `From $x to y$2 billion.`,
			want:  `<p>From $x to y$2 billion.</p>` + "\n",
		},
		{
			name:  "inline display math",
			input: `See $$\sum x$$ here.`,
			want:  `<p>See <span class="math math-display">\sum x</span> here.</p>` + "\n",
		},
		{
			name:  "math block interrupting a paragraph",
			input: "Roughly:\n$$\n\\frac{a}{b}\n$$\nafter",
			want:  "<p>Roughly:</p>\n<div class=\"math math-display\">\\frac{a}{b}\n</div>\n<p>after</p>\n",
		},
		{
			name:  "single line math block",
			input: "$$ a < b $$",
			want:  "<div class=\"math math-display\"> a &lt; b </div>\n",
		},
		{
			name:  "math block and table in blockquote",
			input: "> $$\n> x^2\n> $$\n>\n> | a | b |\n> | - | - |\n> | 1 | 2 |",
			want: "<blockquote>\n<div class=\"math math-display\">x^2\n</div>\n" +
				"<table>\n<thead>\n<tr>\n<th>a</th>\n<th>b</th>\n</tr>\n</thead>\n" +
				"<tbody>\n<tr>\n<td>1</td>\n<td>2</td>\n</tr>\n</tbody>\n</table>\n</blockquote>\n",
		},
		{
			name:  "hashtags",
			input: "#tv #2026 #a/b-c_d #✅/🚧 C#",
			want:  `<p><span class="hashtag">#tv</span> #2026 <span class="hashtag">#a/b-c_d</span> <span class="hashtag">#✅/🚧</span> C#</p>` + "\n",
		},
		{
			name:  "obsidian image",
			input: "![[Pasted image 1.png]]",
			want:  `<p><img src="/images/Pasted image 1.png" alt="/images/Pasted image 1.png"></p>` + "\n",
		},
		{
			name:  "youtube and twitter embeds keep the alt text",
			input: "![Title](https://www.youtube.com/watch?v=abc)\n![](https://youtu.be/def)\n![x.com](https://x.com/u/status/123)",
			want: `<p>Title<iframe width="560" height="315" class="youtube-video" src="https://www.youtube.com/embed/abc" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>` + "\n" +
				`<iframe width="560" height="315" class="youtube-video" src="https://www.youtube.com/embed/def" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>` + "\n" +
				`x.com<blockquote class="twitter-tweet" data-theme="dark"><a href="https://twitter.com/x/status/123"></a></blockquote><script async src="https://platform.twitter.com/widgets.js" charset="utf-8"></script></p>` + "\n",
		},
	}

	md := newMarkdown()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := md.Convert([]byte(tt.input), &buf); err != nil {
				t.Fatal(err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("Convert(%q)\ngot:\n%s\nwant:\n%s", tt.input, got, tt.want)
			}
		})
	}
}
