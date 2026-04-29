# AGENTS.md

## Project Overview

This repository contains the Go static site generator and content for `debugjois.dev`.

## Repository Structure

```text
site/                       # Go static site generator
  content/                  # Source content, including daily notes
  templates/                # HTML templates
  static/                   # Static assets copied into the build output
  cloudfront/               # CloudFront Function source and deploy notes
.github/
  actions/                  # Reusable site workflow actions
  workflows/                # Site CI/deploy workflows
```

## Version Control

This repo is hosted in GitHub, but locally uses [Jujutsu (`jj`)](https://github.com/jj-vcs/jj) for version control. If a `.jj/` directory is present at the root, use `jj` commands for committing, branching, and history operations. Otherwise, fall back to `git`.

## Go Workspace

The repo uses a top-level `go.work` file that includes only:

- `./site`

Use the latest Go version available.

### Source Code Conventions

Use the code conventions of the Go standard library source code. As far as possible and unless specified otherwise minimize third party dependencies.

### Workflow

- After making changes in `site/`, run `golangci-lint run` from `site/`.
- The repo-level `.golangci.yml` enables `gofumpt`, `staticcheck`, `govet`, and `ineffassign` for Go code.
- Do not add separate `go fmt` or `go vet` checks unless there is a specific reason; `golangci-lint` is the source of truth for Go formatting and linting here.

## Site

Run all site commands from `site/`.

### Common commands

- `go build -o debugjois-site .` - build the site binary
- `./debugjois-site build` - build the static site into `build/`
- `./debugjois-site build --dev` - include drafts and scratch content
- `./debugjois-site build --rebuild` - rebuild the entire archive
- `./debugjois-site sync-notes-obsidian` - sync daily notes from Google Drive shared drive
- `./debugjois-site commit-notes` - commit any changes in the daily notes folder
- `./debugjois-site commit-notes --skip-ci` - commit with `[skip ci]` appended to the message
- `./debugjois-site upload` - upload generated files to S3
- `./debugjois-site upload --dryrun` - preview upload without writing to S3
- `./debugjois-site upload --source-dir=<path>` - override source directory
- `./debugjois-site upload --bucket=<name>` - override S3 bucket
- `./debugjois-site build-newsletter` - preview the weekly newsletter
- `./debugjois-site build-newsletter --post` - post newsletter draft to Buttondown
- `./debugjois-site build-newsletter --post --notify` - post and notify via Resend
- `go test ./...` - run all site tests

## CloudFront Function

The source for the live CloudFront Function is in `site/cloudfront/domain-redirect-debugjois-dev.js`. See `site/cloudfront/README.md` for deploy commands.
