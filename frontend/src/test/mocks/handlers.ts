import { http, HttpResponse } from "msw";

// Matches VITE_SITE_BACKEND_URL defined in vitest.config.ts
const BASE_URL = "http://localhost:3000";

export const handlers = [
  // Default happy-path handler. Override per-test with server.use() for error/slow cases.
  http.get(`${BASE_URL}/health`, () =>
    HttpResponse.json({ status: "ok", email: "test@example.com" }),
  ),
];
