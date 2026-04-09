# debugjois.dev

Monorepo for [debugjois.dev](https://debugjois.dev) and authenticated SPA apps under `/app`.

## Layout

- `site/` - Go static site generator for the main website and daily log
- `backend/api/` - Go HTTP API that also runs as an AWS Lambda
- `backend/build-and-push-image.sh` - builds and pushes the Lambda image
- `infra/` - AWS CDK app and deploy script for backend infrastructure
- `frontend/` - Vite + React SPA served at `/app`

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
- `invoke` rejects API Gateway request events locally; use `serve` and send the request over HTTP instead
