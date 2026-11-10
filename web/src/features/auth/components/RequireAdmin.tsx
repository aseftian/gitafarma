import type * as React from "react";
import { ShieldAlert, ShieldCheck } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/components/ui/card";
import { useAuth } from "../auth-context";
import { GoogleSignInButton } from "./GoogleSignInButton";

export function RequireAdmin({ children }: { children: React.ReactNode }) {
  const { status, error, loginWithGoogleIdToken, logout } = useAuth();

  if (status === "loading") {
    return (
      <Card>
        <CardContent className="p-6 text-center text-sm text-muted-foreground">
          Memeriksa sesi admin...
        </CardContent>
      </Card>
    );
  }

  if (status === "forbidden") {
    return (
      <Card className="border-red-200">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-red-700">
            <ShieldAlert className="h-5 w-5" />
            Akses Ditolak
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {error || "Email Google Anda belum terdaftar atau berstatus nonaktif di sheet admins."}
          </p>
          <p className="text-xs text-muted-foreground">
            Pastikan email tercantum pada tab <code>admins</code> dengan kolom <code>active = TRUE</code>, lalu coba masuk kembali.
          </p>
          <Button type="button" variant="outline" onClick={() => void logout()}>
            Ganti Akun Google
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (status === "unauthenticated") {
    return (
      <Card className="mx-auto max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-2 flex h-11 w-11 items-center justify-center rounded-full bg-primary/10 text-primary">
            <ShieldCheck className="h-6 w-6" />
          </div>
          <CardTitle>Login Pengelola Apotek</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 text-center">
          <p className="text-sm text-muted-foreground">
            Masuk menggunakan akun Google yang terdaftar pada sheet <code>admins</code> untuk mengelola daftar obat dan pengaturan.
          </p>
          {error && (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-xs text-red-700">
              {error}
            </div>
          )}
          <GoogleSignInButton onCredential={(tok) => void loginWithGoogleIdToken(tok)} />
        </CardContent>
      </Card>
    );
  }

  return <>{children}</>;
}
