import * as React from "react";
import type { AdminUser } from "@/shared/types/medicine";

export type AuthStatus = "loading" | "authenticated" | "unauthenticated" | "forbidden";

export interface AuthContextValue {
  status: AuthStatus;
  admin: AdminUser | null;
  error: string | null;
  loginWithGoogleIdToken: (idToken: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshSession: () => Promise<void>;
}

export const AuthContext = React.createContext<AuthContextValue | null>(null);

export function useAuth(): AuthContextValue {
  const ctx = React.useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
}
