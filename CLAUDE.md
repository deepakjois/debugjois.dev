# CLAUDE.md

## Project Overview

This is a personal website and daily log application built in Go. The site component lives in the `site/` directory and provides multiple commands for building a static website, syncing daily notes from Obsidian/Google Drive, managing search indexing, uploading to S3, and building newsletters.

The project is structured as a monorepo to support future additions (JavaScript frontend app, backend server) alongside the static site generator.

## Key Commands

All site commands should be run from the `site/` directory.

### Development Commands
- `cd site && go build` - Build the main executable
- `./debugjois.dev build` - Build the static site (outputs to `build/` directory)
- `./debugjois.dev build --dev` - Build in dev mode (includes scratch file and drafts)
- `./debugjois.dev build --rebuild` - Rebuild the entire archive

### Daily Notes Management
- `./debugjois.dev sync-notes-obsidian --obsidian-vault=<path>` - Sync daily notes from Obsidian vault

### Newsletter Commands
- `./debugjois.dev build-newsletter` - Preview weekly newsletter (outputs to stdout)
- `./debugjois.dev build-newsletter --post` - Post newsletter draft to Buttondown
- `./debugjois.dev build-newsletter --post --notify` - Post and send notification email via Resend

`BUTTONDOWN_API_KEY` must be set to post the newsletter. `RESEND_API_KEY` must
be set to send an email. Ask for their values if instructed to run the command
and they havent been provided

### Other Commands
- `./debugjois.dev upload` - Upload files to S3 bucket
- `./watch.sh` - Auto-sync from Obsidian every 60 seconds using viddy

### Testing
- `cd site && go test ./...` - Run all tests
- `go test -v -run TestCalculateNewsletterWeek ./...` - Run specific test

## Architecture

### Project Structure

```
site/                       # Static site generator (Go)
  content/
    daily-notes/            # Markdown files named YYYY-MM-DD.md
      attachments/          # Images and media files
    index.html              # Main page content
  templates/                # HTML templates for different page types
  static/                   # CSS, images, favicon, etc.
  build/                    # Generated static site output
  *.go                      # Go source files
  go.mod / go.sum           # Go module files
.github/
  actions/
    site-setup-and-build/   # Composite action for Go setup and build
  workflows/
    site-build-deploy.yml   # Build and deploy site to S3
    site-govulncheck.yml    # Go vulnerability check
    site-latest-deps.yml    # Test with latest dependencies
    site-newsletter.yml     # Post newsletter to Buttondown
    site-sync-build-deploy.yml  # Sync from GDrive, build, deploy
    claude.yml              # Claude Code integration
```

### Core Components

**Main Application (`site/main.go`)**
- Uses Kong CLI library for command parsing
- Defines all available commands as structs

**Static Site Generator (`site/build.go`)**
- Converts Markdown daily notes to HTML using goldmark
- Supports Obsidian-style features: hashtags, image embeds, and link embeds
- Generates multiple page types: index, daily notes, archive pages, and RSS feed
- Templates stored in `site/templates/` directory, static assets in `site/static/`

**Obsidian Integration**
- Custom goldmark extensions for Obsidian syntax:
  - `ObsidianImageExtender`: Handles `![[image.png]]` syntax
  - `ObsidianEmbedExtender`: Converts YouTube/Twitter URLs to embeds
- Supports hashtag parsing with ObsidianVariant

**Search System (`site/index.go`, `site/search.go`)**
- Uses Bleve full-text search engine
- Indexes all daily notes as plain text (Markdown converted)
- Provides highlighted search results with ANSI colors
- Index stored in `site/debugjois-dev.bleve/` directory

**Content Sync**
- `site/sync_notes_obsidian.go`: Syncs from local Obsidian vault using rsync

### Data Flow

1. Daily notes written in Obsidian or created directly as Markdown files
2. Sync commands pull notes into `site/content/daily-notes/`
3. Build command processes notes through goldmark with custom extensions
4. Generated HTML uses templates to create complete pages
5. Static files and images copied to build directory
6. Optional: Notes indexed for search functionality

## Development Notes

- The application automatically handles Obsidian-style links and embeds
- The build process groups notes by month for archive generation
- RSS feed generation excludes "today's" notes to avoid incomplete entries
- Custom timezone handling via `site/timezone.go` using go-meridian library (currently CET)
- Newsletter week calculation uses ISO week numbers based on Monday (see `site/build_newsletter.go`)
- All Go commands (build, test, etc.) must be run from the `site/` directory
