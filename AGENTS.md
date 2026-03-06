# CLAUDE.md

## Project Overview

This is a personal website and daily log application built in Go. The site component lives in the `site/` directory and provides multiple commands for building a static website, syncing daily notes from Obsidian/Google Drive, managing search indexing, uploading to S3, and building newsletters.

The project is structured as a monorepo to support future additions (JavaScript frontend app, backend server) alongside the static site generator.

## Key Commands

All site commands should be run from the `site/` directory.

### Development Commands
- `cd site && go build` - Build the main executable
- `./debugjois.dev build` - Build the static site (outputs to `build/` directory)
- `./debugjois.dev build --dev` - Build in dev mode (includes scratch file and drafts)
- `./debugjois.dev build --rebuild` - Rebuild the entire archive

### Daily Notes Management
- `./debugjois.dev sync-notes-obsidian --obsidian-vault=<path>` - Sync daily notes from Obsidian vault

### Newsletter Commands
- `./debugjois.dev build-newsletter` - Preview weekly newsletter (outputs to stdout)
- `./debugjois.dev build-newsletter --post` - Post newsletter draft to Buttondown
- `./debugjois.dev build-newsletter --post --notify` - Post and send notification email via Resend

`BUTTONDOWN_API_KEY` must be set to post the newsletter. `RESEND_API_KEY` must
be set to send an email. Ask for their values if instructed to run the command
and they havent been provided

### Other Commands
- `./debugjois.dev upload` - Upload files to S3 bucket
- `./watch.sh` - Auto-sync from Obsidian every 60 seconds using viddy

### Testing
- `cd site && go test ./...` - Run all tests
- `go test -v -run TestCalculateNewsletterWeek ./...` - Run specific test

## Architecture

### Project Structure

```
site/                       # Static site generator (Go)
  content/
    daily-notes/            # Markdown files named YYYY-MM-DD.md
      attachments/          # Images and media files
    index.html              # Main page content
  templates/                # HTML templates for different page types
  static/                   # CSS, images, favicon, etc.
  build/                    # Generated static site output
  *.go                      # Go source files
  go.mod / go.sum           # Go module files
.github/
  actions/
    site-setup-and-build/   # Composite action for Go setup and build
  workflows/
    site-build-deploy.yml   # Build and deploy site to S3
    site-govulncheck.yml    # Go vulnerability check
    site-latest-deps.yml    # Test with latest dependencies
    site-newsletter.yml     # Post newsletter to Buttondown
    site-sync-build-deploy.yml  # Sync from GDrive, build, deploy
    claude.yml              # Claude Code integration
```

### Core Components

**Main Application (`site/main.go`)**
- Uses Kong CLI library for command parsing
- Defines all available commands as structs

**Static Site Generator (`site/build.go`)**
- Converts Markdown daily notes to HTML using goldmark
- Supports Obsidian-style features: hashtags, image embeds, and link embeds
- Generates multiple page types: index, daily notes, archive pages, and RSS feed
- Templates stored in `site/templates/` directory, static assets in `site/static/`

**Obsidian Integration**
- Custom goldmark extensions for Obsidian syntax:
  - `ObsidianImageExtender`: Handles `![[image.png]]` syntax
  - `ObsidianEmbedExtender`: Converts YouTube/Twitter URLs to embeds
- Supports hashtag parsing with ObsidianVariant

**Search System (`site/index.go`, `site/search.go`)**
- Uses Bleve full-text search engine
- Indexes all daily notes as plain text (Markdown converted)
- Provides highlighted search results with ANSI colors
- Index stored in `site/debugjois-dev.bleve/` directory

**Content Sync**
- `site/sync_notes_obsidian.go`: Syncs from local Obsidian vault using rsync

### Data Flow

1. Daily notes written in Obsidian or created directly as Markdown files
2. Sync commands pull notes into `site/content/daily-notes/`
3. Build command processes notes through goldmark with custom extensions
4. Generated HTML uses templates to create complete pages
5. Static files and images copied to build directory
6. Optional: Notes indexed for search functionality

## Development Notes

- The application automatically handles Obsidian-style links and embeds
- The build process groups notes by month for archive generation
- RSS feed generation excludes "today's" notes to avoid incomplete entries
- Custom timezone handling via `site/timezone.go` using go-meridian library (currently CET)
- Newsletter week calculation uses ISO week numbers based on Monday (see `site/build_newsletter.go`)
- All Go commands (build, test, etc.) must be run from the `site/` directory

## Backend (AWS Lambda)

A FastAPI-based Lambda backend lives in `backend/`:

```
backend/
  infra/        # AWS CDK project in Go — defines all AWS infrastructure
  api/
    app/
      main.py   # FastAPI app + Mangum handler + get_email_from_request helper
    tests/
      test_main.py
    pyproject.toml
    Dockerfile
```

The app is dual-mode: `handler = Mangum(app)` is used by Lambda; `__main__` runs a local uvicorn server.

### Local development (run from `backend/api/`)
```bash
uv run python -m app.main   # starts uvicorn at http://localhost:8000 with --reload
uv run pytest -v            # run tests
```

### Prerequisites
- Docker Desktop must be running for `cdk deploy` (it builds the image locally)
- AWS credentials must be available in the default profile (`aws sts get-caller-identity` to verify)

### CDK Commands (run from `backend/infra/`)
- `cdk diff` - preview infrastructure changes
- `cdk deploy --require-approval never` - build image, push to ECR, deploy stack
- `cdk synth` - emit CloudFormation template without deploying

### API Gateway
An HTTP API (v2) fronts the Lambda. JWT auth is enforced at the gateway level — the app itself has no auth middleware. User email is extracted from `request.scope["aws.event"]` JWT claims (returns `None` locally). CORS allows any origin. The API URL is printed as `ApiUrl` after deploy.

### Adding API dependencies (run from `backend/api/`)
```bash
uv add <package>   # adds to pyproject.toml and updates uv.lock
```

## Frontend (Vite + React)

A React SPA lives in `frontend/`, served under the `/logger` base path. Built with Vite, TanStack Router (file-based routing), and TanStack Query.

```
frontend/
  src/
    auth.ts               # AuthContext, AuthState, useAuth hook
    main.tsx              # Entry point — QueryClient, Router, providers
    routes/
      __root.tsx          # Root layout — auth gate + dev bypass
      index.tsx           # Index route (/logger/)
    routeTree.gen.ts      # Auto-generated by TanStack Router plugin (gitignored)
  index.html
  vite.config.ts
  .env                    # Production env vars (gitignored — real AWS URL + Google client ID)
  .env.example            # Template
  .env.development        # Dev env vars: VITE_AUTH_BYPASS=true, localhost backend URL
```

### Commands (run from `frontend/`)
- `npm run dev` — start dev server (serves at http://localhost:5173/logger/)
- `npm run build` — production build (outputs to `../site/build/logger/`)
- `npm run preview` — preview production build locally (must `npm run build` first; go to http://localhost:4173/logger/)
- `npm test` — run all tests once (CI mode)
- `npm run test:watch` — interactive watch mode
- `npm run test:coverage` — run tests with coverage report
- `npm run lint` — oxlint
- `npm run fmt` — oxfmt (format all files)

### Configuration
- Vite `base: '/logger/'` ensures all asset paths are prefixed for S3 sub-folder deployment
- TanStack Router `basepath: '/logger'` aligns client-side routing with the base path
- `VITE_SITE_BACKEND_URL` sets the backend API origin — `http://localhost:8000` in dev, full AWS URL in prod
- `VITE_AUTH_BYPASS=true` in `.env.development` skips Google login and uses a fake `"dev"` token; never set in production
- Auth token is persisted to `localStorage` under key `logger_auth_token`; Google One Tap (`auto_select: true`) silently refreshes it on mount

### Local full-stack development
Run both servers concurrently:
```bash
# Terminal 1 — backend
cd backend/api && uv run python -m app.main

# Terminal 2 — frontend
cd frontend && npm run dev
```
Visit http://localhost:5173/logger/ — no login required in dev mode. API calls go directly to http://localhost:8000 (CORS allowed by the backend middleware).

### Testing
Test infrastructure uses Vitest + Testing Library + MSW. Key files:
- `vitest.config.ts` — separate from `vite.config.ts`; omits the TanStack Router plugin (which generates `routeTree.gen.ts`) so tests never depend on it
- `src/test/utils.tsx` — `renderWithRouter`, `createTestQueryClient`, `makePreAuthenticatedRoot`
- `src/test/mocks/handlers.ts` — default MSW handlers; override per-test with `server.use()`
- `src/__tests__/` — test files

**Patterns for adding new tests:**

- **Auth gate** (route should not render unless logged in): pass `RootComponent` as `rootComponent`; mock `@react-oauth/google`; click the mock login button to simulate auth
  ```tsx
  vi.mock('@react-oauth/google', () => ({ ... }))
  import { RootComponent } from '../routes/__root'
  await renderWithRouter({ rootComponent: RootComponent })
  ```

- **Inner route** (bypassing the login gate): use `makePreAuthenticatedRoot(AuthContext)` — provides a real `useAuth()` context with a fake token
  ```tsx
  import { AuthContext } from '../auth'
  const PreAuthRoot = makePreAuthenticatedRoot(AuthContext)
  await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: MyRoute })
  ```

- **Network mocking** (TanStack Query renders): MSW intercepts fetch at the network level; override the default handler inside a test for error/slow cases
  ```tsx
  server.use(http.get('http://localhost:3000/endpoint', () => new HttpResponse(null, { status: 503 })))
  ```

- **Adding a route**: pass the exported component and its path pattern to `renderWithRouter`
  ```tsx
  import { MyRoute } from '../routes/my-route'
  await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: MyRoute, pathPattern: '/my-route' })
  ```
