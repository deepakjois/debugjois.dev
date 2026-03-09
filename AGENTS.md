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

## Go Workspace

The repo uses a top-level `go.work` file that includes:

- `./site`
- `./backend/api`
- `./infra`

All Go modules use Go `1.26.1`.

## Site

Run all site commands from `site/`.

### Common commands

- `go build -o debugjois-site .` - build the site binary
- `./debugjois-site build` - build the static site into `build/`
- `./debugjois-site build --dev` - include drafts and scratch content
- `./debugjois-site build --rebuild` - rebuild the entire archive
- `./debugjois-site sync-notes-obsidian --obsidian-vault=<path>` - sync daily notes from Obsidian
- `./debugjois-site upload` - upload generated files to S3
- `./debugjois-site build-newsletter` - preview the weekly newsletter
- `./debugjois-site build-newsletter --post` - post newsletter draft to Buttondown
- `./debugjois-site build-newsletter --post --notify` - post and notify via Resend
- `go test ./...` - run all site tests

### Notes

- `BUTTONDOWN_API_KEY` is required for newsletter posting
- `RESEND_API_KEY` is required for notification email sending
- `watch.sh` wraps `sync-notes-obsidian` with `viddy`
- templates live in `site/templates/`, static assets in `site/static/`

## Backend API

The backend API lives in `backend/api/` and is written in Go.

### Behavior

- runs as a normal HTTP server locally
- runs as an AWS Lambda when `AWS_LAMBDA_RUNTIME_API` is set
- currently serves `/` and `/health`
- reads authenticated user email from API Gateway JWT context when invoked via Lambda

### Common commands

Run these from `backend/api/`:

- `go run .` - start the local server on `http://localhost:8000`
- `PORT=9000 go run .` - override the local port
- `go test ./...` - run backend tests
- `go build .` - build the binary

### Image build

From the repository root:

- `./backend/build-and-push-image.sh` - build and push the Lambda image, then print an immutable `IMAGE_URI`

Prerequisites:

- Docker Desktop must be running
- AWS credentials must be available in the default profile

## Infrastructure

The CDK app lives in `infra/`.

### Common commands

Run these from `infra/` unless the command already includes the path:

- `cdk diff` - preview infrastructure changes
- `IMAGE_URI=<ecr-image-uri-or-digest> cdk deploy --require-approval never` - deploy with an explicit image
- `cdk synth` - synthesize the CloudFormation template
- `./infra/deploy.sh` - deploy using the image currently configured on the deployed Lambda
- `./infra/deploy.sh --build-image` - build and push a new image first, then deploy

### Notes

- `infra/infra.go` falls back to the currently deployed Lambda image when `IMAGE_URI` is unset; set `IMAGE_URI` explicitly for deploys that should change the image
- `infra/deploy.sh` calls `../backend/build-and-push-image.sh` when `--build-image` is used
- the API URL is emitted as the `ApiUrl` stack output after deploy
- `infra/cloudfront/domain-redirect-debugjois-dev.js` is the source for the production CloudFront Function that redirects the apex domain and rewrites `/app` SPA routes

## Frontend

The frontend lives in `frontend/` and is a Vite + React SPA served under `/app`.

### Commands

Run these from `frontend/`:

- `npm run dev` - start the dev server at `http://localhost:5173/app/`
- `npm run build` - build to `../site/build/app/`
- `npm run preview` - preview the production build locally
- `npm test` - run tests once
- `npm run test:watch` - run tests in watch mode
- `npm run test:coverage` - generate coverage output
- `npm run lint` - run oxlint
- `npm run fmt` - format with oxfmt

### Configuration

- Vite `base` is `/app/`
- TanStack Router `basepath` is `/app`
- `VITE_SITE_BACKEND_URL` points at the backend API origin
- `VITE_GOOGLE_CLIENT_ID` configures Google login
- `VITE_AUTH_BYPASS=true` can be set in `.env.development` to bypass login in local dev

### Local full-stack development

```bash
# Terminal 1
cd backend/api && go run .

# Terminal 2
cd frontend && npm run dev
```

## GitHub Workflows

Current workflows live in `.github/workflows/`:

- `site-build-deploy.yml`
- `site-govulncheck.yml`
- `site-latest-deps.yml`
- `site-newsletter.yml`
- `claude.yml`
