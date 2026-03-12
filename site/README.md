# Site

Static site generator for the main `debugjois.dev` website and daily log.

## Requirements

- Go 1.26+
- optional: `viddy` for `watch.sh`
- AWS credentials for `upload`

## Setup

Run from `site/`:

```bash
go build -o debugjois-site .
./debugjois-site --help
```

## Common commands

```bash
./debugjois-site build
./debugjois-site build --dev
./debugjois-site build --rebuild
./debugjois-site sync-notes-obsidian --obsidian-vault=<path>
./debugjois-site sync-notes-to-gdrive --folder-id=<id> --creds=<service-account.json>
./debugjois-site sync-notes-to-gdrive --folder-id=<id> --creds=<service-account.json> --dry-run
./debugjois-site sync-notes-to-gdrive --folder-id=<id> --creds=<service-account.json> --all
./debugjois-site upload
./debugjois-site build-newsletter
./debugjois-site build-newsletter --post
./debugjois-site build-newsletter --post --notify
go test ./...
```

## Content layout

- `content/daily-notes/` - daily Markdown notes named `YYYY-MM-DD.md`
- `content/daily-notes/attachments/` - images and other note assets
- `content/index.html` - homepage content
- `templates/` - page templates
- `static/` - static assets copied into the build output

## Notes

- newsletter posting requires `BUTTONDOWN_API_KEY`
- notification emails require `RESEND_API_KEY`
- Google Drive note sync uses `GOOGLE_DRIVE_FOLDER_ID` and `GOOGLE_DRIVE_CREDENTIALS_FILE` if you prefer env vars
- Google Drive note sync only considers top-level daily note files matching `YYYY-MM-DD.md`; it ignores attachments and other files
- Google Drive note sync processes only the 30 most recent notes by default; use `--all` to include all matching notes
- `watch.sh` runs `sync-notes-obsidian` via `viddy`
