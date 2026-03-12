package main

import (
	"github.com/alecthomas/kong"
)

var CLI struct {
	Build             BuildCmd             `cmd:"" help:"Build the static site"`
	SyncNotesObsidian SyncNotesObsidianCmd `cmd:"" help:"Sync daily notes from Obsidian vault"`
	SyncNotesToGdrive SyncNotesToGdriveCmd `cmd:"" help:"Sync daily note files to Google Drive"`
	Upload            UploadCmd            `cmd:"" help:"Upload files to S3 bucket"`
	BuildNewsletter   BuildNewsletterCmd   `cmd:"" help:"Build weekly newsletter from daily notes"`
}

func main() {
	ctx := kong.Parse(&CLI)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
