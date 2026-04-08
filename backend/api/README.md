# Backend API

Go backend API for `debugjois.dev`.

## Modes

- local HTTP server for development
- AWS Lambda runtime when `AWS_LAMBDA_RUNTIME_API` is set

## Endpoints

- `GET /` - returns a simple greeting payload (also used as healthcheck)
- `GET /daily` - load today's daily note from Google Drive
- `POST /daily` - save today's daily note to Google Drive
- `GET /linkpreview` - proxy to LinkPreview API
- `POST /podcast-transcribe` - parse Podcast Addict input and trigger podcast transcription

## Requirements

- Go 1.26+
- Docker Desktop for container builds
- AWS credentials for image push and deploy steps

## Local development

Run from `backend/api/`:

```bash
cat > .env <<'EOF'
LINKPREVIEW_API_KEY=your-linkpreview-api-key
DEEPGRAM_API_KEY=your-deepgram-api-key
EOF

# Configure Google Drive access (one-time)
gcloud auth application-default login \
  --impersonate-service-account=gdrive-obsidian@daily-notes-obsidian-gdrive.iam.gserviceaccount.com \
  --scopes=https://www.googleapis.com/auth/drive

go run .
```

The server listens on `http://localhost:8000` by default.

Local startup loads environment variables from `.env` and requires both
`LINKPREVIEW_API_KEY` and `DEEPGRAM_API_KEY` to be present there. Google Drive
access uses Application Default Credentials (ADC).

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

## Standalone transcription CLI

Run from `backend/api/`:

```bash
go run ./cmd/transcribe "<podcast-addict-share-text-or-url>"
```

The CLI reads `DEEPGRAM_API_KEY` from `backend/api/.env`, parses the Podcast
Addict episode metadata, sends the episode audio URL to Deepgram, and prints the
transcript JSON to stdout.

To also store the transcript JSON in S3, pass a bucket ARN with `--store`:

```bash
go run ./cmd/transcribe --store arn:aws:s3:::debugjois-dev-site \
  "<podcast-addict-share-text-or-url>"
```

When `--store` is set, the CLI also exports `TRANSCRIPT_BUCKET_ARN` for the
process and writes the same transcript JSON to `transcripts/<stable-name>.json`
in the specified bucket.

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
