package main

import (
	"github.com/alecthomas/kong"
)

var CLI struct {
	Build             BuildCmd             `cmd:"" help:"Build the static site"`
	SyncNotesObsidian SyncNotesObsidianCmd `cmd:"" help:"Sync daily notes from Obsidian vault"`
	SyncNotesGdrive   SyncNotesGdriveCmd   `cmd:"" help:"Sync daily notes from Google Drive"`
	Upload            UploadCmd            `cmd:"" help:"Upload files to S3 bucket"`
	BuildNewsletter   BuildNewsletterCmd   `cmd:"" help:"Build weekly newsletter from daily notes"`
	Index             IndexCmd             `cmd:"" help:"Index daily notes"`
	Search            SearchCmd            `cmd:"" help:"Search indexed daily notes"`
}

func main() {
	ctx := kong.Parse(&CLI)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
