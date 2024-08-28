package main

import (
	"github.com/alecthomas/kong"
)

var CLI struct {
	Build     BuildCmd     `cmd:"" help:"Build the static site"`
	SyncNotes SyncNotesCmd `cmd:"" help:"Sync daily notes from Obsidian vault"`
}

func main() {
	ctx := kong.Parse(&CLI)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
