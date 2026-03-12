import { describe, it, expect, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";

vi.mock("@react-oauth/google", () => ({
  GoogleOAuthProvider: ({ children }: { children: React.ReactNode }) => children,
  GoogleLogin: () => null,
}));

import { renderWithRouter, makePreAuthenticatedRoot } from "../test/utils";
import { AuthContext } from "../auth";
import { Index } from "../routes/index";
import { Logger } from "../routes/logger";
import { Podscriber } from "../routes/podscriber";
import { latestDailyNote } from "../test/mocks/handlers";
import { server } from "../test/mocks/server";

// Bypasses the login gate; provides a real AuthContext value so useAuth() succeeds.
const PreAuthRoot = makePreAuthenticatedRoot(AuthContext);

describe("Index route - app launcher", () => {
  it("renders links to available apps", async () => {
    await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: Index });

    expect(screen.getByRole("link", { name: "Open Logger" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open Podscriber" })).toBeInTheDocument();
  });
});

describe("Logger route - markdown editor", () => {
  it("renders the editor shell with daily note content", async () => {
    await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: Logger });

    await waitFor(() => expect(screen.queryByText("Loading editor...")).not.toBeInTheDocument());
    expect(screen.getByRole("heading", { name: "2026-03-12.md" })).toBeInTheDocument();
    expect(screen.getByRole("textbox")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    expect(screen.getByText("Daily Note")).toBeInTheDocument();
    expect(screen.getByText("Test content.")).toBeInTheDocument();
  });

  it("shows an unauthorized message for a forbidden backend response", async () => {
    const ForbiddenRoot = makePreAuthenticatedRoot(AuthContext);

    server.use(
      http.get("http://localhost:3000/daily", () => new HttpResponse(null, { status: 403 })),
    );

    await renderWithRouter({ rootComponent: ForbiddenRoot, routeComponent: Logger });

    expect(
      screen.getByText("Unauthorized access. Sign in with an approved account."),
    ).toBeInTheDocument();
  });

  it("saves the edited note and returns to a clean state", async () => {
    const user = userEvent.setup();
    await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: Logger });

    await waitFor(() => expect(screen.queryByText("Loading editor...")).not.toBeInTheDocument());
    const editor = screen.getByRole("textbox");
    expect(editor).toHaveClass("cm-lineWrapping");
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();

    await user.click(editor);
    await user.keyboard("\nExtra line");

    await waitFor(() => expect(screen.getByRole("button", { name: "Save" })).toBeEnabled());

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "Saved" })).toBeDisabled());
    expect(screen.getByRole("textbox")).toHaveClass("cm-lineWrapping");
    expect(latestDailyNote.contents).toContain("Extra line");
  });
});

describe("Podscriber route - placeholder", () => {
  it("renders placeholder content", async () => {
    await renderWithRouter({ rootComponent: PreAuthRoot, routeComponent: Podscriber });

    expect(screen.getByRole("heading", { name: "Podscriber" })).toBeInTheDocument();
    expect(screen.getByText("Placeholder for the next app.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to apps" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign out" })).toBeInTheDocument();
  });
});
