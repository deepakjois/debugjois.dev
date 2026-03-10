import { http, HttpResponse } from "msw";

// Matches VITE_SITE_BACKEND_URL defined in vitest.config.ts
const BASE_URL = "http://localhost:3000";

export const handlers = [
  http.get(`${BASE_URL}/health`, ({ request }) => {
    const authHeader = request.headers.get("Authorization");

    if (!authHeader) {
      return HttpResponse.json({ error: "unauthorized" }, { status: 401 });
    }

    if (authHeader === "Bearer wrong-user-token") {
      return HttpResponse.json({ error: "forbidden" }, { status: 403 });
    }

    return HttpResponse.json({ status: "ok", email: "test@example.com" });
  }),
];
