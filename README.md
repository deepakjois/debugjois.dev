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
- Backend API: `cd backend/api && go test ./... && go run .`
- Frontend: `cd frontend && npm install && npm run dev`
- Infra deploy: `./infra/deploy.sh --build-image`
