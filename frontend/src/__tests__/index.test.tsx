import { describe, it, expect, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";

vi.mock("@react-oauth/google", () => ({
  GoogleOAuthProvider: ({ children }: { children: React.ReactNode }) => children,
  GoogleLogin: () => null,
}));

import { server } from "../test/mocks/server";
import { renderWithRouter, makePreAuthenticatedRoot } from "../test/utils";
import { AuthContext } from "../auth";
import { Index } from "../routes/index";

// Bypasses the login gate; provides a real AuthContext value so useAuth() succeeds.
const PreAuthRoot = makePreAuthenticatedRoot(AuthContext);

describe("Index route - health check", () => {
  it("renders the Check Health button in the idle state", async () => {
    await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: Index });

    const button = screen.getByRole("button", { name: "Check Health" });
    expect(button).toBeInTheDocument();
    expect(button).not.toBeDisabled();
    expect(screen.queryByText(/Backend status/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Error/)).not.toBeInTheDocument();
  });

  it("shows loading state while the request is in flight", async () => {
    server.use(
      // Never resolves — simulates a slow/hung network request
      http.get("http://localhost:3000/health", () => new Promise(() => undefined)),
    );

    const user = userEvent.setup();
    await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: Index });

    await user.click(screen.getByRole("button", { name: "Check Health" }));

    expect(screen.getByRole("button", { name: "Checking…" })).toBeDisabled();
  });

  it("displays backend status on successful response", async () => {
    // Default handler returns { status: 'ok', email: 'test@example.com' }
    const user = userEvent.setup();
    await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: Index });

    await user.click(screen.getByRole("button", { name: "Check Health" }));

    await waitFor(() =>
      expect(screen.getByText("Backend status: ok (user: test@example.com)")).toBeInTheDocument(),
    );
  });

  it("displays error message on HTTP failure", async () => {
    server.use(
      http.get("http://localhost:3000/health", () => new HttpResponse(null, { status: 503 })),
    );

    const user = userEvent.setup();
    await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: Index });

    await user.click(screen.getByRole("button", { name: "Check Health" }));

    await waitFor(() => expect(screen.getByText(/Error: HTTP 503/)).toBeInTheDocument());
  });

  it("sends Authorization header with the auth token", async () => {
    let capturedAuth: string | null = null;

    server.use(
      http.get("http://localhost:3000/health", ({ request }) => {
        capturedAuth = request.headers.get("Authorization");
        return HttpResponse.json({ status: "ok" });
      }),
    );

    const user = userEvent.setup();
    await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: Index });

    await user.click(screen.getByRole("button", { name: "Check Health" }));

    await waitFor(() => expect(capturedAuth).toBe("Bearer fake-test-token"));
  });
});
