# debugjois.dev

Static site generator and content for [debugjois.dev](https://debugjois.dev).

## Layout

- `site/` - Go static site generator for the main website and daily log
- `site/content/` - source content, including daily notes
- `site/templates/` - HTML templates
- `site/static/` - static assets copied into the build output
- `site/cloudfront/` - source and deploy notes for the live CloudFront Function

## Workspace

The Go workspace in `go.work` includes only `./site`.

## Common commands

Run from `site/` unless noted otherwise:

- `go build -o debugjois-site .` - build the site binary
- `./debugjois-site build` - build the static site into `build/`
- `./debugjois-site build --dev` - include drafts and scratch content
- `./debugjois-site build --rebuild` - rebuild the entire archive
- `./debugjois-site sync-notes-obsidian` - sync daily notes from Google Drive shared drive
- `./debugjois-site commit-notes` - commit daily note changes
- `./debugjois-site commit-notes --skip-ci` - commit with `[skip ci]` appended to the message
- `./debugjois-site upload` - upload generated files to S3
- `./debugjois-site upload --dryrun` - preview upload without writing to S3
- `./debugjois-site build-newsletter` - preview the weekly newsletter
- `./debugjois-site build-newsletter --post` - post newsletter draft to Buttondown
- `./debugjois-site build-newsletter --post --notify` - post and notify via Resend
- `go test ./...` - run site tests
- `golangci-lint run` - run the configured Go lint/format checks

## Deployments

- Site deploys are handled by the site GitHub workflows and `./debugjois-site upload`.
- CloudFront Function source is tracked in `site/cloudfront/`; see `site/cloudfront/README.md` for manual deploy commands.
