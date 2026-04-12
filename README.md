# debugjois.dev

Monorepo for [debugjois.dev](https://debugjois.dev) and frontend apps under `/apps`.

## Layout

- `site/` - Go static site generator for the main website and daily log
- `backend/api/` - Go HTTP API that also runs as an AWS Lambda
- `backend/build-and-push-image.sh` - builds and pushes the Lambda image
- `infra/` - AWS CDK app and deploy script for backend infrastructure
- `frontend/` - Vite frontend apps served under `/apps`

## Workspace

Go code is split across `site/`, `backend/api/`, and `infra/`, with a shared
workspace in `go.work`. All Go modules use Go `1.26.1`.

## Common commands

- Site: `cd site && go build -o debugjois-site . && ./debugjois-site build`
- Backend API: `cd backend/api && go test ./... && go run . serve`
- Backend invoke: `cd backend/api && printf '{"action":"health-check"}' | go run . invoke`
- Frontend: `cd frontend && npm install && npm run dev`
- Infra deploy: `./infra/deploy.sh`
- Infra deploy with fresh image: `./infra/deploy.sh --build-image`

## Backend local usage

- Start the local API server: `cd backend/api && go run . serve`
- Invoke the shared event handler with JSON from stdin: `cd backend/api && printf '{"action":"health-check"}' | go run . invoke`
- Invoke with a payload file: `cd backend/api && go run . invoke --payload event.json`
- Transcribe a Podcast Addict episode by piping the share text or URL over stdin: `cd backend/api && printf '%s\n' 'https://podcastaddict.com/example/episode/123' | go run ./cmd/podcast-transcribe`
- Prefer stdin piping over positional arguments for `podcast-transcribe`, especially for Markdown-formatted or multiline share text, to avoid shell quoting issues
- `invoke` rejects API Gateway request events locally; use `serve` and send the request over HTTP instead
