"use client";

import { useEffect, useRef, useState } from "react";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";
import { Alert } from "@/components/ui/Alert";
import {
  generateGoogleNonce,
  GOOGLE_CLIENT_ID,
  loadGoogleGsiScript,
} from "@/lib/google-auth";
import { cn } from "@/lib/utils";

export interface SocialAuthButtonsProps {
  mode?: "login" | "register" | "reauth";
  showDivider?: boolean;
  disabled?: boolean;
  onCredential: (payload: {
    idToken: string;
    nonce: string;
  }) => void | Promise<void>;
  onError?: (code: string) => void;
}

const MIN_GSI_WIDTH = 200;
const MAX_GSI_WIDTH = 400;
const DEFAULT_GSI_WIDTH = 320;

function clampGsiWidth(rawWidth: number): number {
  if (!Number.isFinite(rawWidth) || rawWidth <= 0) {
    return DEFAULT_GSI_WIDTH;
  }
  return Math.max(MIN_GSI_WIDTH, Math.min(MAX_GSI_WIDTH, Math.floor(rawWidth)));
}

export function SocialAuthButtons({
  mode = "login",
  showDivider,
  disabled = false,
  onCredential,
  onError,
}: SocialAuthButtonsProps) {
  const { language, hydrated, t } = useI18n();
  const { theme } = useTheme();

  const shouldShowDivider = showDivider ?? mode !== "reauth";

  const wrapperRef = useRef<HTMLDivElement | null>(null);
  const buttonHostRef = useRef<HTMLDivElement | null>(null);
  const rawNonceRef = useRef<string | null>(null);
  const onCredentialRef = useRef(onCredential);
  const onErrorRef = useRef(onError);
  const disabledRef = useRef(disabled);

  const [nonceVersion, setNonceVersion] = useState(0);
  const [containerWidth, setContainerWidth] =
    useState<number>(DEFAULT_GSI_WIDTH);
  const [loadState, setLoadState] = useState<"loading" | "ready" | "error">(
    "loading"
  );

  useEffect(() => {
    onCredentialRef.current = onCredential;
    onErrorRef.current = onError;
    disabledRef.current = disabled;
  }, [disabled, onCredential, onError]);

  useEffect(() => {
    if (!GOOGLE_CLIENT_ID) return;
    const el = wrapperRef.current;
    if (!el) return;

    const updateWidth = (rawWidth: number) => {
      const next = clampGsiWidth(rawWidth);
      setContainerWidth((prev) => (Math.abs(prev - next) >= 2 ? next : prev));
    };

    updateWidth(el.getBoundingClientRect().width);

    if (typeof ResizeObserver === "undefined") {
      return;
    }

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (!entry) return;
      updateWidth(entry.contentRect.width);
    });
    observer.observe(el);

    return () => {
      observer.disconnect();
    };
  }, []);

  useEffect(() => {
    if (!GOOGLE_CLIENT_ID || !hydrated) return;
    let cancelled = false;
    const hostEl = buttonHostRef.current;

    async function setupGoogleButton() {
      try {
        const [{ rawNonce, hashedNonce }] = await Promise.all([
          generateGoogleNonce(),
          loadGoogleGsiScript(language),
        ]);

        if (cancelled) return;

        const gsi = window.google?.accounts?.id;
        const host = buttonHostRef.current;
        if (!gsi || !host) {
          throw new Error("Google Identity Services is unavailable");
        }

        rawNonceRef.current = rawNonce;

        gsi.initialize({
          client_id: GOOGLE_CLIENT_ID,
          nonce: hashedNonce,
          ux_mode: "popup",
          context: mode === "register" ? "signup" : "signin",
          auto_select: false,
          callback: (response) => {
            if (cancelled || disabledRef.current) return;

            const idToken = response?.credential;
            const currentNonce = rawNonceRef.current;
            rawNonceRef.current = null;
            setNonceVersion((v) => v + 1);

            if (!idToken || !currentNonce) {
              onErrorRef.current?.("INVALID_CREDENTIALS");
              return;
            }

            void onCredentialRef.current({
              idToken,
              nonce: currentNonce,
            });
          },
        });

        host.replaceChildren();
        gsi.renderButton(host, {
          type: "standard",
          theme: theme === "dark" ? "filled_black" : "outline",
          size: "large",
          shape: "pill",
          text: mode === "register" ? "signup_with" : "continue_with",
          locale: language,
          width: containerWidth,
        });

        if (!cancelled) {
          setLoadState("ready");
        }
      } catch {
        if (!cancelled) {
          setLoadState("error");
          if (mode === "reauth") {
            onErrorRef.current?.("OAUTH_UNAVAILABLE");
          }
        }
      }
    }

    void setupGoogleButton();

    return () => {
      cancelled = true;
      rawNonceRef.current = null;
      hostEl?.replaceChildren();
      window.google?.accounts?.id?.cancel();
    };
  }, [containerWidth, hydrated, language, mode, nonceVersion, theme]);

  if (!GOOGLE_CLIENT_ID) {
    if (mode === "reauth") {
      return (
        <Alert variant="error">{t.profile.deleteDialogGoogleUnavailable}</Alert>
      );
    }
    return null;
  }

  return (
    <div
      ref={wrapperRef}
      aria-disabled={disabled || undefined}
      className={cn(
        "w-full space-y-4 shadow-none",
        shouldShowDivider ? "pt-1" : "pt-0",
        disabled && "pointer-events-none opacity-60"
      )}
    >
      {shouldShowDivider ? (
        <div className="flex items-center gap-3">
          <div className="h-px flex-1 bg-border" />
          <span className="shrink-0 text-xs uppercase tracking-wider text-muted-foreground">
            {t.login.orDivider}
          </span>
          <div className="h-px flex-1 bg-border" />
        </div>
      ) : null}

      <div className="flex min-h-[44px] w-full flex-col items-center justify-center">
        <div
          ref={buttonHostRef}
          className={cn(
            "flex w-full justify-center shadow-none",
            loadState !== "ready" && "hidden"
          )}
        />

        {loadState === "loading" ? (
          <div className="flex h-11 w-full max-w-[400px] items-center justify-center rounded-full border border-border bg-muted/40 px-4 text-xs text-muted-foreground shadow-none">
            {t.login.googleLoading}
          </div>
        ) : null}

        {loadState === "error" ? (
          mode === "reauth" ? (
            <Alert variant="error">{t.login.googleSignInError}</Alert>
          ) : (
            <p className="text-center text-xs text-muted-foreground">
              {t.login.googleSignInError}
            </p>
          )
        ) : null}
      </div>
    </div>
  );
}
