# debugjois.dev

Monorepo for [debugjois.dev](https://debugjois.dev) — a personal website and daily log. Contains the static site generator (`site/`), a Lambda backend (`backend/`), and a React frontend (`frontend/`).

Go code is organized as multiple modules in `site/`, `backend/api/`, and `backend/infra/`, with a shared workspace in `go.work` so editor tooling and local multi-module workflows work cleanly. All Go modules use Go `1.26.1`.
