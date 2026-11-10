import { useEffect, useRef, useState } from "react";

interface GoogleAccountsId {
  initialize: (config: {
    client_id: string;
    callback: (response: { credential?: string }) => void;
  }) => void;
  renderButton: (
    parent: HTMLElement,
    options: {
      theme?: "outline" | "filled_blue" | "filled_black";
      size?: "large" | "medium" | "small";
      text?: "signin_with" | "signup_with" | "continue_with";
      shape?: "rectangular" | "pill";
      width?: number;
    },
  ) => void;
}

declare global {
  interface Window {
    google?: {
      accounts?: {
        id?: GoogleAccountsId;
      };
    };
  }
}

interface GoogleSignInButtonProps {
  onCredential: (idToken: string) => void;
}

export function GoogleSignInButton({ onCredential }: GoogleSignInButtonProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const onCredentialRef = useRef(onCredential);
  onCredentialRef.current = onCredential;

  const clientId = import.meta.env.VITE_GOOGLE_CLIENT_ID ?? "";
  const [scriptError, setScriptError] = useState<string | null>(null);

  useEffect(() => {
    if (!clientId) return;
    let cancelled = false;

    const renderGoogleButton = () => {
      const gsi = window.google?.accounts?.id;
      const el = containerRef.current;
      if (!gsi || !el || cancelled) return;

      el.innerHTML = "";
      gsi.initialize({
        client_id: clientId,
        callback: (resp) => {
          if (resp.credential) {
            onCredentialRef.current(resp.credential);
          }
        },
      });
      gsi.renderButton(el, {
        theme: "outline",
        size: "large",
        text: "signin_with",
        shape: "rectangular",
        width: 280,
      });
    };

    if (window.google?.accounts?.id) {
      renderGoogleButton();
      return;
    }

    const existingScript = document.querySelector<HTMLScriptElement>(
      'script[src="https://accounts.google.com/gsi/client"]',
    );
    if (existingScript) {
      existingScript.addEventListener("load", renderGoogleButton);
      return () => {
        cancelled = true;
        existingScript.removeEventListener("load", renderGoogleButton);
      };
    }

    const script = document.createElement("script");
    script.src = "https://accounts.google.com/gsi/client";
    script.async = true;
    script.defer = true;
    script.onload = renderGoogleButton;
    script.onerror = () => {
      if (!cancelled) {
        setScriptError("Gagal memuat skrip Google Identity Services.");
      }
    };
    document.head.appendChild(script);

    return () => {
      cancelled = true;
    };
  }, [clientId]);

  if (!clientId) {
    return (
      <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800">
        <code>VITE_GOOGLE_CLIENT_ID</code> belum dikonfigurasi pada environment web.
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div ref={containerRef} className="flex min-h-[44px] items-center justify-center" />
      {scriptError && <p className="text-center text-xs text-destructive">{scriptError}</p>}
    </div>
  );
}
