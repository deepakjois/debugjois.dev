# Frontend

React SPA for the `/logger` experience.

## Stack

- Vite
- React 19
- TanStack Router
- TanStack Query
- Vitest + Testing Library + MSW

## Requirements

- Node.js
- npm

## Commands

Run from `frontend/`:

```bash
npm run dev
npm run build
npm run preview
npm test
npm run test:watch
npm run test:coverage
npm run lint
npm run fmt
```

## Routing and build output

- Vite `base` is `/logger/`
- TanStack Router `basepath` is `/logger`
- production builds are written to `../site/build/logger/`

## Environment

Use `.env.example` as a starting point.

- `VITE_SITE_BACKEND_URL` - backend API origin
- `VITE_GOOGLE_CLIENT_ID` - Google OAuth client ID
- `VITE_AUTH_BYPASS=true` in `.env.development` - bypass login in local dev

## Local development

Frontend only:

```bash
npm run dev
```

Full stack:

```bash
# Terminal 1
cd backend/api && go run .

# Terminal 2
cd frontend && npm run dev
```

Open `http://localhost:5173/logger/`.

## Testing notes

- tests run without the TanStack Router Vite plugin
- `src/test/` contains shared test utilities and MSW setup
- `src/__tests__/` contains route-level tests
