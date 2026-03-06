import { useState } from "react";
import { createRootRoute, Outlet } from "@tanstack/react-router";
import {
  GoogleLogin,
  useGoogleOneTapLogin,
  googleLogout,
  type CredentialResponse,
} from "@react-oauth/google";
import { AuthContext } from "../auth";

const STORAGE_KEY = "logger_auth_token";

export function RootComponent() {
  const [token, setToken] = useState<string | null>(
    () => import.meta.env.VITE_AUTH_BYPASS === "true" ? "dev" : localStorage.getItem(STORAGE_KEY),
  );

  function handleCredential(credential: string) {
    localStorage.setItem(STORAGE_KEY, credential);
    setToken(credential);
  }

  function handleSignOut() {
    localStorage.removeItem(STORAGE_KEY);
    googleLogout();
    setToken(null);
  }

  // Runs once on mount. With auto_select: true, Google silently signs in the
  // user if they still have an active Google session — no UI shown.
  // disabled: !!token avoids an unnecessary prompt when already authenticated.
  useGoogleOneTapLogin({
    onSuccess: (res: CredentialResponse) => {
      if (res.credential) handleCredential(res.credential);
    },
    auto_select: true,
    disabled: import.meta.env.VITE_AUTH_BYPASS === "true" || !!token,
  });

  if (!token) {
    return (
      <div style={{ padding: "2rem" }}>
        <h1>Logger</h1>
        <p>Sign in to continue.</p>
        <GoogleLogin
          onSuccess={(res: CredentialResponse) => {
            if (res.credential) handleCredential(res.credential);
          }}
          onError={() => console.error("Google login failed")}
        />
      </div>
    );
  }

  return (
    <AuthContext.Provider value={{ token }}>
      <div style={{ padding: "2rem" }}>
        <header
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: "1rem",
          }}
        >
          <h1 style={{ margin: 0 }}>Logger</h1>
          {import.meta.env.VITE_AUTH_BYPASS !== "true" && (
            <button onClick={handleSignOut}>Sign out</button>
          )}
        </header>
        <Outlet />
      </div>
    </AuthContext.Provider>
  );
}

export const Route = createRootRoute({
  component: RootComponent,
});
