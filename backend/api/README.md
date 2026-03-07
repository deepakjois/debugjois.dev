# Backend API

This directory contains the Go backend API for `debugjois.dev`.

It supports two modes:
- local HTTP server for development
- AWS Lambda container runtime when `AWS_LAMBDA_RUNTIME_API` is set

## Requirements

- Go 1.26+
- Docker Desktop for container builds
- AWS credentials for push/deploy steps

## Local development

Run the API locally:

```bash
go run .
```

The server listens on `http://localhost:8000` by default.

Override the port with:

```bash
PORT=9000 go run .
```

## Tests

Run all tests:

```bash
go test ./...
```

## Build

Build the local binary:

```bash
go build .
```

## Docker

Build the Lambda container image locally:

```bash
docker build -t debugjois-dev-api .
```

The Docker image uses a multi-stage build and a minimal `scratch` final image.

## Deploy

From the repository root:

```bash
./backend/build-and-push-image.sh
./backend/deploy.sh --build-image
```
