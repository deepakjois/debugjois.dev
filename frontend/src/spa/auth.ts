import { createContext, useContext } from "react";

export interface AuthState {
  token: string;
  signOut: () => void;
}

export const AuthContext = createContext<AuthState | null>(null);

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within an authenticated route");
  return ctx;
}
