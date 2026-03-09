import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { CredentialResponse } from "@react-oauth/google";

// vi.hoisted creates a value alongside vi.mock hoisting, giving test code
// access to the One Tap callback after the component has registered it.
const oneTap = vi.hoisted(() => ({
  callback: null as ((res: CredentialResponse) => void) | null,
}));

vi.mock("@react-oauth/google", () => ({
  GoogleOAuthProvider: ({ children }: { children: React.ReactNode }) => children,
  GoogleLogin: ({
    onSuccess,
  }: {
    onSuccess: (res: CredentialResponse) => void;
    onError: () => void;
  }) => (
    <button
      data-testid="mock-google-login"
      onClick={() => onSuccess({ credential: "fake-credential-token" } as CredentialResponse)}
    >
      Sign in with Google
    </button>
  ),
  useGoogleOneTapLogin: (opts: { onSuccess: (res: CredentialResponse) => void }) => {
    oneTap.callback = opts.onSuccess;
  },
  googleLogout: vi.fn(),
}));

import { RootComponent } from "../routes/__root";
import { Logger } from "../routes/logger";
import { renderWithRouter } from "../test/utils";

beforeEach(() => {
  localStorage.clear();
  oneTap.callback = null;
  vi.clearAllMocks();
});

describe("RootComponent - auth gate", () => {
  it("shows login UI when unauthenticated", async () => {
    await renderWithRouter({ rootComponent: RootComponent });

    expect(screen.getByText("Sign in to continue.")).toBeInTheDocument();
    expect(screen.getByTestId("mock-google-login")).toBeInTheDocument();
    expect(screen.queryByTestId("route-content")).not.toBeInTheDocument();
  });

  it("shows inner route content after successful login", async () => {
    const user = userEvent.setup();
    await renderWithRouter({ rootComponent: RootComponent });

    await user.click(screen.getByTestId("mock-google-login"));

    expect(screen.getByTestId("route-content")).toBeInTheDocument();
    expect(screen.queryByTestId("mock-google-login")).not.toBeInTheDocument();
  });

  it("does not render shared app chrome when authenticated", async () => {
    const user = userEvent.setup();
    await renderWithRouter({ rootComponent: RootComponent });

    await user.click(screen.getByTestId("mock-google-login"));

    expect(screen.queryByRole("button", { name: "Sign out" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Apps" })).not.toBeInTheDocument();
  });

  it("returns to login screen after route-level sign out", async () => {
    const { googleLogout } = await import("@react-oauth/google");
    const user = userEvent.setup();
    await renderWithRouter({
      rootComponent: RootComponent,
      routeComponent: Logger,
      initialEntry: "/logger",
      pathPattern: "/logger",
    });

    await user.click(screen.getByTestId("mock-google-login"));
    await user.click(screen.getByRole("button", { name: "Sign out" }));

    expect(screen.getByTestId("mock-google-login")).toBeInTheDocument();
    expect(localStorage.getItem("app_auth_token")).toBeNull();
    expect(googleLogout).toHaveBeenCalledOnce();
  });
});

describe("RootComponent - localStorage persistence", () => {
  it("shows authenticated UI on mount when token is in localStorage", async () => {
    localStorage.setItem("app_auth_token", "stored-token");

    await renderWithRouter({ rootComponent: RootComponent });

    expect(screen.getByTestId("route-content")).toBeInTheDocument();
    expect(screen.queryByTestId("mock-google-login")).not.toBeInTheDocument();
  });

  it("saves token to localStorage after login", async () => {
    const user = userEvent.setup();
    await renderWithRouter({ rootComponent: RootComponent });

    await user.click(screen.getByTestId("mock-google-login"));

    expect(localStorage.getItem("app_auth_token")).toBe("fake-credential-token");
  });
});

describe("RootComponent - One Tap silent refresh", () => {
  it("authenticates silently when One Tap fires on mount", async () => {
    await renderWithRouter({ rootComponent: RootComponent });

    expect(screen.getByTestId("mock-google-login")).toBeInTheDocument();

    await act(async () => {
      oneTap.callback?.({ credential: "one-tap-token" } as CredentialResponse);
    });

    expect(screen.getByTestId("route-content")).toBeInTheDocument();
    expect(localStorage.getItem("app_auth_token")).toBe("one-tap-token");
  });

  it("updates localStorage when One Tap refreshes an existing token", async () => {
    localStorage.setItem("app_auth_token", "old-token");
    await renderWithRouter({ rootComponent: RootComponent });

    await act(async () => {
      oneTap.callback?.({ credential: "refreshed-token" } as CredentialResponse);
    });

    expect(localStorage.getItem("app_auth_token")).toBe("refreshed-token");
  });
});
