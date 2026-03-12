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

## Site

Run all site commands from `site/`.

### Common commands

- `go build -o debugjois-site .` - build the site binary
- `./debugjois-site build` - build the static site into `build/`
- `./debugjois-site build --dev` - include drafts and scratch content
- `./debugjois-site build --rebuild` - rebuild the entire archive
- `./debugjois-site sync-notes-obsidian --obsidian-vault=<path>` - sync daily notes from Obsidian
- `./debugjois-site sync-notes-obsidian --obsidian-vault=<path> --no-git` - sync without committing
- `./debugjois-site upload` - upload generated files to S3
- `./debugjois-site upload --dryrun` - preview upload without writing to S3
- `./debugjois-site upload --source-dir=<path>` - override source directory
- `./debugjois-site upload --bucket=<name>` - override S3 bucket
- `./debugjois-site build-newsletter` - preview the weekly newsletter
- `./debugjois-site build-newsletter --post` - post newsletter draft to Buttondown
- `./debugjois-site build-newsletter --post --notify` - post and notify via Resend
- `go test ./...` - run all site tests

### Environment variables

| Variable | Required | Description |
|---|---|---|
| `BUTTONDOWN_API_KEY` | For newsletter posting | Buttondown API key |
| `RESEND_API_KEY` | For notification emails | Resend API key |

### Notes

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

- After making changes under `frontend/`, run `npm run fmt` and `npm run lint` before finishing.
- Then run the narrowest relevant verification command for the change, usually `npm test` or `npm run build`.

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
| `site-build-deploy.yml` | Daily cron (23:01 UTC) + manual | Build and deploy static site to S3 |
| `site-test-and-deploy.yml` | Push to `site/**` on main | Run site tests, build, and deploy to S3 |
| `site-govulncheck.yml` | Push to `site/**` on main | Run `govulncheck` on site module |
| `site-latest-deps.yml` | Scheduled | Update site Go dependencies |
| `site-newsletter.yml` | Weekly cron (Sundays 2am UTC) | Post weekly newsletter to Buttondown |
| `backend-api-test.yml` | Push to `backend/api/**` on main | Run backend Go tests |
| `frontend-test-deploy.yml` | Push to `frontend/**` on main | Run frontend tests and deploy to S3 |
| `claude.yml` | Issue/PR comment with `@claude` | Claude Code bot integration |
