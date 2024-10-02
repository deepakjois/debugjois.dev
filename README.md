Website: https://debugjois.dev

Daily Log: https://debugjois.dev/daily

## Code
* Go and `html/template` package for the website.
* `sync-notes-obsidian` subcommand syncs daily notes from a local Obsidian repo
* `sync-notes-gdrive` subcommand syncs daily notes from a Google Drive folder
* `build` subcommand builds the site (`watch.sh` watches for changes and builds the site).
* `upload` subcommand uploads the site to S3, where it is picked up by Cloudfront and served on the debujois.dev domain.
