# AGENTS.md

## Project Overview

This repository is a monorepo for `debugjois.dev`. It contains:

- a Go static site generator in `site/`
- a Go backend API in `backend/api/`
- AWS CDK infrastructure in `infra/`
- a React frontend in `frontend/`

## Repository Structure

```text
site/                       # Go static site generator
  content/                  # Source content, including daily notes
  templates/                # HTML templates
  static/                   # Static assets
backend/
  api/                      # Go HTTP API / Lambda handler
  build-and-push-image.sh   # Build and push backend container image
infra/                      # AWS CDK app and deploy script
  cloudfront/               # CloudFront Function source files
frontend/                   # Vite + React SPA for /app
.github/
  actions/
  workflows/
```

## Version Control

This repo is hosted in Github, but locally uses [Jujutsu (`jj`)](https://github.com/jj-vcs/jj) for version control. If a `.jj/` directory is present at the root, use `jj` commands for committing, branching, and history operations. Otherwise, fall back to `git`. If it is jj repo, it is most likely a workspace, in which case the colocated git repo is in the default workspace.

## Go Workspace

The repo uses a top-level `go.work` file that includes:

- `./site`
- `./backend/api`
- `./infra`

Use the latest Go version available.

### Source Code Conventions
Use the code conventions of the Go standard library source code. As far as possible and unless specified otherwise minimize third party dependencies.

### Workflow

- After making changes in any Go module, run `golangci-lint run` from that module directory.
- The repo-level `.golangci.yml` enables `gofumpt`, `staticcheck`, `govet`, and `ineffassign` for all Go code in `site/`, `backend/api/`, and `infra/`.
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

## Backend API

The backend API lives in `backend/api/` and is written in Go.

### Common commands

Run these from `backend/api/`:

- `go run .` - start the local server on `http://localhost:8000`
- `PORT=9000 go run .` - override the local port
- `printf '{"action":"health-check"}' | go run . invoke` - invoke the shared backend event handler with event JSON from stdin
- `go test ./...` - run backend tests
- `go build .` - build the binary

## Infrastructure

The CDK app lives in `infra/`.

### Common commands

Run these from `infra/` unless the command already includes the path:

- `cdk diff` - preview infrastructure changes
- `cdk --app 'go mod download && go run infra.go --image-uri <ecr-image-uri-or-digest>' deploy --require-approval never` - deploy with an explicit image
- `cdk synth` - synthesize the CloudFormation template
- `./infra/deploy.sh` - deploy using the image currently configured on the deployed Lambda
- `./infra/deploy.sh --build-image` - build and push a new image first, then deploy

## Frontend

The frontend lives in `frontend/` and is a Vite + React SPA served under `/app`.

### Commands

Run these from `frontend/`:

- `npm run dev` - start the dev server at `http://localhost:5173/app/`
- `npm run dev:prod-env` - dev server using production env vars
- `npm run build` - build to `../site/build/app/`
- `npm run preview` - preview the production build locally
- `npm test` - run tests once
- `npm run test:watch` - run tests in watch mode
- `npm run test:coverage` - generate coverage output
- `npm run lint` - run oxlint
- `npm run lint:fix` - auto-fix linting issues
- `npm run fmt` - format with oxfmt
- `npm run fmt:check` - check formatting without writing

### Workflow

After every set of frontend edits, always run these steps in order — no exceptions, even for small changes:

1. `npm run fmt` — format all files (must run first; reformats code in place)
2. `npm run lint` — check for lint errors
3. `npm run build` — default final check for frontend TypeScript changes; this runs `tsc -b` and catches type errors that tests and lint can miss
4. `npm test` — also run when the change affects runtime behavior, component behavior, routing behavior, or test-covered functionality

### Configuration

- Vite `base` is `/app/`
- TanStack Router `basepath` is `/app`
- Copy `frontend/.env.example` to `frontend/.env` as a starting point

### Local full-stack development

```bash
# Terminal 1
cd backend/api && go run .

# Terminal 2
cd frontend && npm run dev
```

### Github Actions Convention
- Keep workflow YAML files declarative. Do not inline multiline or complex bash in `run:` blocks — extract any non-trivial shell logic into a script under `.github/scripts/` and call it from the workflow step.
