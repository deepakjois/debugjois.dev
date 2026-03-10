import { useEffect, useState } from "react";
import { createRootRoute, Outlet } from "@tanstack/react-router";
import {
  GoogleLogin,
  useGoogleOneTapLogin,
  googleLogout,
  type CredentialResponse,
} from "@react-oauth/google";
import { AuthContext } from "../auth";

const STORAGE_KEY = "app_auth_token";
const API_URL = import.meta.env.VITE_SITE_BACKEND_URL;

type AuthStatus = "checking" | "ready" | "unauthenticated" | "forbidden" | "error";

export function RootComponent() {
  const bypassAuth = import.meta.env.VITE_AUTH_BYPASS === "true";
  const initialToken = bypassAuth ? "dev" : localStorage.getItem(STORAGE_KEY);
  const [token, setToken] = useState<string | null>(initialToken);
  const [authStatus, setAuthStatus] = useState<AuthStatus>(
    bypassAuth ? "ready" : initialToken ? "checking" : "unauthenticated",
  );
  const [authMessage, setAuthMessage] = useState<string | null>(null);

  function handleCredential(credential: string) {
    localStorage.setItem(STORAGE_KEY, credential);
    setToken(credential);
    setAuthStatus("checking");
    setAuthMessage(null);
  }

  function handleSignOut() {
    localStorage.removeItem(STORAGE_KEY);
    googleLogout();
    setToken(null);
    setAuthStatus("unauthenticated");
    setAuthMessage(null);
  }

  useEffect(() => {
    if (bypassAuth) {
      setAuthStatus("ready");
      setAuthMessage(null);
      return;
    }

    if (!token) {
      if (authStatus === "checking") {
        setAuthStatus("unauthenticated");
      }
      return;
    }

    if (authStatus !== "checking") {
      return;
    }

    const controller = new AbortController();

    async function validateToken() {
      try {
        const res = await fetch(`${API_URL}/health`, {
          signal: controller.signal,
          headers: { Authorization: `Bearer ${token}` },
        });

        if (res.ok) {
          setAuthStatus("ready");
          return;
        }

        localStorage.removeItem(STORAGE_KEY);
        setToken(null);

        if (res.status === 403) {
          setAuthStatus("forbidden");
          setAuthMessage("Unauthorized access. Sign in with an approved account.");
          return;
        }

        setAuthStatus("unauthenticated");
        setAuthMessage(null);
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") {
          return;
        }

        setAuthStatus("error");
        setAuthMessage("Could not reach the backend.");
      }
    }

    void validateToken();

    return () => controller.abort();
  }, [authStatus, bypassAuth, token]);

  useGoogleOneTapLogin({
    onSuccess: (res: CredentialResponse) => {
      if (res.credential) handleCredential(res.credential);
    },
    disabled: bypassAuth || authStatus === "checking" || !!token,
  });

  if (authStatus !== "ready") {
    return (
      <div className="app-auth-page">
        <div className="app-auth-shell">
          <div className="app-auth-copy">
            <p className="app-auth-eyebrow">debugjois.dev apps</p>
            <h1 className="app-auth-title">
              {authStatus === "checking" ? "Checking sign-in..." : "Sign in to continue."}
            </h1>
            {authMessage ? <p className="app-auth-message">{authMessage}</p> : null}
          </div>
          {authStatus !== "checking" ? (
            <div className="app-auth-action">
              <GoogleLogin
                onSuccess={(res: CredentialResponse) => {
                  if (res.credential) handleCredential(res.credential);
                }}
                onError={() => console.error("Google login failed")}
              />
            </div>
          ) : null}
        </div>
      </div>
    );
  }

  const readyToken = token;

  if (!readyToken) {
    return null;
  }

  return (
    <AuthContext.Provider value={{ token: readyToken, signOut: handleSignOut }}>
      <Outlet />
    </AuthContext.Provider>
  );
}

export const Route = createRootRoute({
  component: RootComponent,
});
