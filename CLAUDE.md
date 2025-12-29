# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a personal website and daily log application built in Go. The main executable `debugjois.dev` provides multiple commands for building a static website, syncing daily notes from Obsidian/Google Drive, managing search indexing, uploading to S3, and building newsletters.

## Key Commands

### Development Commands
- `go build` - Build the main executable
- `./debugjois.dev build` - Build the static site (outputs to `build/` directory)
- `./debugjois.dev build --dev` - Build in dev mode (includes scratch file and drafts)
- `./debugjois.dev build --rebuild` - Rebuild the entire archive

### Daily Notes Management
- `./debugjois.dev sync-notes-obsidian --obsidian-vault=<path>` - Sync daily notes from Obsidian vault
- `./debugjois.dev sync-notes-gdrive --folder-id=<id> --creds=<path>` - Sync from Google Drive
- `./debugjois.dev index` - Create/update search index for daily notes
- `./debugjois.dev search <query>` - Search indexed daily notes with highlighted results

### Newsletter Commands
- `./debugjois.dev build-newsletter` - Preview weekly newsletter (outputs to stdout)
- `./debugjois.dev build-newsletter --post` - Post newsletter draft to Buttondown
- `./debugjois.dev build-newsletter --post --notify` - Post and send notification email via Resend

### Other Commands
- `./debugjois.dev upload` - Upload files to S3 bucket
- `./watch.sh` - Auto-sync from Obsidian every 60 seconds using viddy

### Testing
- `go test ./...` - Run all tests
- `go test -v -run TestCalculateNewsletterWeek ./...` - Run specific test

## Architecture

### Core Components

**Main Application (`main.go`)**
- Uses Kong CLI library for command parsing
- Defines all available commands as structs

**Static Site Generator (`build.go`)**
- Converts Markdown daily notes to HTML using goldmark
- Supports Obsidian-style features: hashtags, image embeds, and link embeds
- Generates multiple page types: index, daily notes, archive pages, and RSS feed
- Templates stored in `templates/` directory, static assets in `static/`

**Obsidian Integration**
- Custom goldmark extensions for Obsidian syntax:
  - `ObsidianImageExtender`: Handles `![[image.png]]` syntax
  - `ObsidianEmbedExtender`: Converts YouTube/Twitter URLs to embeds
- Supports hashtag parsing with ObsidianVariant

**Search System (`index.go`, `search.go`)**
- Uses Bleve full-text search engine
- Indexes all daily notes as plain text (Markdown converted)
- Provides highlighted search results with ANSI colors
- Index stored in `debugjois-dev.bleve/` directory

**Content Sync**
- `sync_notes_obsidian.go`: Syncs from local Obsidian vault using rsync
- `sync_notes_gdrive.go`: Syncs from Google Drive
- Both commands include git operations (pull, commit, push) unless `--no-git` flag used

### Directory Structure

```
content/
  daily-notes/           # Markdown files named YYYY-MM-DD.md
    attachments/         # Images and media files
  index.html            # Main page content
templates/              # HTML templates for different page types
static/                 # CSS, images, favicon, etc.
build/                  # Generated static site output
```

### Data Flow

1. Daily notes written in Obsidian or created directly as Markdown files
2. Sync commands pull notes into `content/daily-notes/`
3. Build command processes notes through goldmark with custom extensions
4. Generated HTML uses templates to create complete pages
5. Static files and images copied to build directory
6. Optional: Notes indexed for search functionality

## Development Notes

- The application automatically handles Obsidian-style links and embeds
- Conflict files from Google Drive sync are automatically skipped
- The build process groups notes by month for archive generation
- RSS feed generation excludes "today's" notes to avoid incomplete entries
- Custom timezone handling via `timezone.go` using go-meridian library (currently CET)
- Newsletter week calculation uses ISO week numbers based on Monday (see `build_newsletter.go`)