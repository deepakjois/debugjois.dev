Website: https://debugjois.dev

Daily Log: https://debugjois.dev/daily

## Code
* Go and `html/template` package for the website.
* `sync-notes.sh` rsyncs the daily notes from my Obsidian repo and puts them in `content/daily-notes`
* `watch.sh` watches for changes and compiles the Go code and builds the site.
* `upload.sh` uploads the site to S3, where it is picked up by Cloudfront and served on the debujois.dev domain.
