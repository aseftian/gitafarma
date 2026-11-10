import * as React from "react";
import { ApiError, setCsrfToken } from "@/shared/api/client";
import type { AdminUser } from "@/shared/types/medicine";
import { fetchCurrentAdmin, loginWithGoogle, logoutAdmin } from "./api";
import { AuthContext, type AuthContextValue, type AuthStatus } from "./auth-context";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = React.useState<AuthStatus>("loading");
  const [admin, setAdmin] = React.useState<AdminUser | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  const refreshSession = React.useCallback(async () => {
    try {
      const res = await fetchCurrentAdmin();
      setCsrfToken(res.csrf_token);
      setAdmin(res.admin);
      setError(null);
      setStatus("authenticated");
    } catch (err: unknown) {
      setAdmin(null);
      if (err instanceof ApiError && err.status === 403) {
        setError(err.message);
        setStatus("forbidden");
        return;
      }
      setError(null);
      setStatus("unauthenticated");
    }
  }, []);

  React.useEffect(() => {
    void refreshSession();
  }, [refreshSession]);

  const loginWithGoogleIdToken = React.useCallback(async (idToken: string) => {
    setError(null);
    try {
      const res = await loginWithGoogle(idToken);
      setCsrfToken(res.csrf_token);
      setAdmin(res.admin);
      setStatus("authenticated");
    } catch (err: unknown) {
      setAdmin(null);
      if (err instanceof ApiError && err.status === 403) {
        setError(err.message);
        setStatus("forbidden");
        return;
      }
      setError(err instanceof Error ? err.message : "Login Google gagal");
      setStatus("unauthenticated");
    }
  }, []);

  const logout = React.useCallback(async () => {
    try {
      await logoutAdmin();
    } finally {
      setCsrfToken("");
      setAdmin(null);
      setError(null);
      setStatus("unauthenticated");
    }
  }, []);

  const value = React.useMemo<AuthContextValue>(
    () => ({
      status,
      admin,
      error,
      loginWithGoogleIdToken,
      logout,
      refreshSession,
    }),
    [status, admin, error, loginWithGoogleIdToken, logout, refreshSession],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
