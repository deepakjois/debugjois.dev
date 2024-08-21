package main

import (
	"github.com/alecthomas/kong"
)

var CLI struct {
	Build BuildCmd `cmd:"" help:"Build the static site"`
}

func main() {
	ctx := kong.Parse(&CLI)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
