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

This repo uses [Jujutsu (`jj`)](https://github.com/jj-vcs/jj) for version control. The `.jj/` directory is present at the root. Use `jj` commands (not `git`) for committing, branching, and history operations.

## Go Workspace

The repo uses a top-level `go.work` file that includes:

- `./site`
- `./backend/api`
- `./infra`

All Go modules use Go `1.26.1`.

### Workflow

- After making changes in any Go module, run `golangci-lint run` from that module directory.
- The repo-level `.golangci.yml` enables `gofumpt` and `staticcheck` for all Go code in `site/`, `backend/api/`, and `infra/`.
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
- `golangci-lint run` - run Go linting with `gofumpt` and `staticcheck`
- `go test ./...` - run all site tests

### Environment variables

| Variable | Required | Description |
|---|---|---|
| `BUTTONDOWN_API_KEY` | For newsletter posting | Buttondown API key |
| `RESEND_API_KEY` | For notification emails | Resend API key |
| `OBSIDIAN_SHARED_DRIVE` | No (default: `obsidian`) | Name of the Google Drive shared drive |
| `OBSIDIAN_VAULT_FOLDER` | No (default: `PersonalKnowledgeWiki`) | Vault folder name within the shared drive |

### Notes

- `sync-notes-obsidian` pulls daily notes from a Google Drive shared drive using Application Default Credentials (ADC); configure via `gcloud auth application-default login --impersonate-service-account=<sa-email> --scopes=https://www.googleapis.com/auth/drive`
- `commit-notes` stages and commits changes under `content/daily-notes/`; the commit message includes a timestamp and optionally `[skip ci]`
- templates live in `site/templates/`, static assets in `site/static/`

## Backend API

The backend API lives in `backend/api/` and is written in Go.

### Behavior

- runs as a normal HTTP server locally
- runs as an AWS Lambda when `AWS_LAMBDA_RUNTIME_API` is set
- currently serves `/` (also used as the healthcheck endpoint)
- reads authenticated user email from API Gateway JWT context when invoked via Lambda

### Common commands

Run these from `backend/api/`:

- `go run .` - start the local server on `http://localhost:8000`
- `PORT=9000 go run .` - override the local port
- `golangci-lint run` - run Go linting with `gofumpt` and `staticcheck`
- `go test ./...` - run backend tests
- `go build .` - build the binary

### Environment variables

| Variable | Required | Description |
|---|---|---|
| `GITHUB_TOKEN` | Yes for local dev | GitHub PAT loaded from `backend/api/.env` when running locally |
| `PORT` | No (default: `8000`) | HTTP server port for local dev |
| `AWS_LAMBDA_RUNTIME_API` | In Lambda only | Set automatically by Lambda runtime; switches server to Lambda mode |

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
- `cdk --app 'go mod download && go run infra.go --image-uri <ecr-image-uri-or-digest>' deploy --require-approval never` - deploy with an explicit image
- `cdk synth` - synthesize the CloudFormation template
- `golangci-lint run` - run Go linting with `gofumpt` and `staticcheck`
- `./infra/deploy.sh` - deploy using the image currently configured on the deployed Lambda
- `./infra/deploy.sh --build-image` - build and push a new image first, then deploy

### Environment variables

| Variable | Required | Description |
|---|---|---|
| `AWS_ROLE_ARN` | For CI | IAM role ARN assumed by GitHub Actions via OIDC |

### Notes

- `infra/infra.go` falls back to the currently deployed Lambda image when no `--image-uri` argument is provided
- `infra/deploy.sh` calls `../backend/build-and-push-image.sh` when `--build-image` is used and passes the resulting image URI directly to `infra.go`
- the API URL is emitted as the `ApiUrl` stack output after deploy
- `infra/cloudfront/domain-redirect-debugjois-dev.js` is the source for the production CloudFront Function that redirects the apex domain and rewrites `/app` SPA routes
- the CDK stack creates a Secrets Manager secret named `debugjois-dev/github-pat`; set or rotate its value outside CDK with `aws secretsmanager update-secret --secret-id debugjois-dev/github-pat --secret-string '<github-pat>'`
- Lambda receives `GITHUB_PAT_SECRET_ARN`, reads the PAT from Secrets Manager during startup, and sets `GITHUB_TOKEN` in-process

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

### Environment variables

| Variable | Required | Description |
|---|---|---|
| `VITE_SITE_BACKEND_URL` | Yes | Backend API origin |
| `VITE_GOOGLE_CLIENT_ID` | Yes | Google OAuth client ID |
| `VITE_AUTH_BYPASS` | No | Set to `true` in `.env.development` to skip login in local dev |

### Local full-stack development

```bash
# Terminal 1
cd backend/api && go run .

# Terminal 2
cd frontend && npm run dev
```

## GitHub Workflows

Workflows live in `.github/workflows/`:

| Workflow | Trigger | Description |
|---|---|---|
| `site-sync-build-deploy.yml` | Every 15 min + manual | Sync notes from Google Drive, build and deploy static site to S3 |
| `go-lint.yml` | Push to Go paths on main | Run `golangci-lint` with `gofumpt` and `staticcheck` for `site/`, `backend/api/`, and `infra/` |
| `site-test-and-deploy.yml` | Push to `site/**` on main | Run site tests, build, and deploy to S3 |
| `site-govulncheck.yml` | Push to `site/**` on main | Run `govulncheck` on site module |
| `site-latest-deps.yml` | Scheduled | Update site Go dependencies |
| `site-newsletter.yml` | Weekly cron (Sundays 2am UTC) | Post weekly newsletter to Buttondown |
| `backend-api-test-deploy.yml` | Push to `backend/api/**` on main + manual | Run backend tests and deploy |
| `infra-deploy.yml` | Manual | Deploy infra without rebuilding the backend image |
| `frontend-test-deploy.yml` | Push to `frontend/**` on main | Run frontend tests and deploy to S3 |
| `claude.yml` | Issue/PR comment with `@claude` | Claude Code bot integration |
