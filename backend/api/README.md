# Backend API

Go backend API for `debugjois.dev`.

## Modes

- local HTTP server for development
- AWS Lambda runtime when `AWS_LAMBDA_RUNTIME_API` is set

## Endpoints

- `GET /` - returns a simple greeting payload
- `GET /health` - returns `{ "status": "ok" }` and includes user email when present in Lambda JWT context

## Requirements

- Go 1.26+
- Docker Desktop for container builds
- AWS credentials for image push and deploy steps

## Local development

Run from `backend/api/`:

```bash
go run .
```

The server listens on `http://localhost:8000` by default.

To override the port:

```bash
PORT=9000 go run .
```

## Tests

```bash
go test ./...
```

## Build

```bash
go build .
```

## Docker image

From the repository root:

```bash
./backend/build-and-push-image.sh
```

That script builds the Lambda image from `backend/api/`, pushes it to ECR, and
prints an immutable `IMAGE_URI`.

## Deploy

From the repository root:

```bash
./infra/deploy.sh --build-image
```

That command builds and pushes a new image, then passes the resulting immutable
image URI directly to `infra.go` during `cdk deploy`.

The CDK app and deploy script live in the top-level `infra/` directory.
