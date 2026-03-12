import { http, HttpResponse } from "msw";

// Matches VITE_SITE_BACKEND_URL defined in vitest.config.ts
const BASE_URL = "http://localhost:3000";

function authCheck(request: Request) {
  const authHeader = request.headers.get("Authorization");
  if (!authHeader) return HttpResponse.json({ error: "unauthorized" }, { status: 401 });
  if (authHeader === "Bearer wrong-user-token")
    return HttpResponse.json({ error: "forbidden" }, { status: 403 });
  return null;
}

export const handlers = [
  http.get(`${BASE_URL}/`, ({ request }) => {
    return authCheck(request) ?? HttpResponse.json({ status: "ok", email: "test@example.com" });
  }),

  http.get(`${BASE_URL}/daily`, ({ request }) => {
    return (
      authCheck(request) ??
      HttpResponse.json({
        title: "2026-03-12.md",
        contents: btoa("# Daily Note\n\nTest content."),
      })
    );
  }),
];
