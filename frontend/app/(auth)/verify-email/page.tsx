"use client";

import { Suspense, useEffect, useRef, useState, type FormEvent } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { FieldError } from "@/components/ui/FieldError";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AuthCardSkeleton } from "@/components/ui/PageSkeletons";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";
import { focusFieldById } from "@/lib/form-validation";
import { cn } from "@/lib/utils";

function VerifyEmailContent() {
  const searchParams = useSearchParams();
  const { language, t } = useI18n();
  const initialToken = searchParams.get("token") ?? "";

  const [token, setToken] = useState(initialToken);
  const [tokenError, setTokenError] = useState<string | null>(null);
  const [tokenDirty, setTokenDirty] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [verified, setVerified] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const autoVerifiedRef = useRef(false);

  useEffect(() => {
    if (!initialToken || autoVerifiedRef.current) {
      return;
    }
    autoVerifiedRef.current = true;
    window.history.replaceState(null, "", window.location.pathname);

    async function runVerification() {
      setSubmitting(true);
      setError(null);
      try {
        await apiClient.verifyEmail(initialToken);
        setVerified(true);
      } catch (err) {
        const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
        setError(getFriendlyErrorMessage(code, language));
      } finally {
        setSubmitting(false);
      }
    }

    void runVerification();
  }, [initialToken, language]);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);

    if (!token.trim()) {
      setTokenError(t.common.tokenRequired);
      focusFieldById("verify-token");
      return;
    }

    setTokenError(null);
    setSubmitting(true);

    try {
      await apiClient.verifyEmail(token.trim());
      setVerified(true);
    } catch (err) {
      const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
      setError(getFriendlyErrorMessage(code, language));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Card className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5 sm:p-7">
      <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground">
        {t.verifyEmail.title}
      </h1>

      {verified ? (
        <div className="space-y-4">
          <Alert variant="success">{t.verifyEmail.successMessage}</Alert>
          <Button
            render={<Link href="/login" />}
            nativeButton={false}
            className="h-11 rounded-full px-5 text-sm shadow-none"
          >
            {t.verifyEmail.goToLogin}
          </Button>
        </div>
      ) : (
        <>
          {error ? <Alert variant="error">{error}</Alert> : null}
          {submitting && token ? (
            <p className="text-sm text-muted-foreground">
              {t.verifyEmail.verifying}
            </p>
          ) : null}

          <form
            noValidate
            onSubmit={(e) => void handleSubmit(e)}
            className="space-y-4"
          >
            <div className="space-y-1.5">
              <Label
                htmlFor="verify-token"
                className="text-sm font-medium text-foreground"
              >
                {t.verifyEmail.tokenLabel}
              </Label>
              <Input
                id="verify-token"
                name="token"
                type="text"
                autoCorrect="off"
                autoCapitalize="none"
                spellCheck={false}
                required
                disabled={submitting}
                value={token}
                onChange={(e) => {
                  const next = e.target.value;
                  setToken(next);
                  setTokenDirty(true);
                  if (tokenError && next.trim()) {
                    setTokenError(null);
                  }
                }}
                onBlur={() => {
                  if (tokenDirty && !token.trim()) {
                    setTokenError(t.common.tokenRequired);
                  }
                }}
                aria-invalid={Boolean(tokenError)}
                aria-describedby={tokenError ? "verify-token-error" : undefined}
                placeholder={t.verifyEmail.tokenPlaceholder}
                className={cn(
                  "h-11 rounded-2xl bg-card px-3.5 font-mono text-sm shadow-none",
                  tokenError &&
                    "border-[var(--error-text)] focus-visible:border-[var(--error-text)]"
                )}
              />
              <FieldError id="verify-token-error" message={tokenError} />
            </div>

            <Button
              type="submit"
              disabled={submitting}
              className="h-11 w-full rounded-full text-sm font-medium shadow-none"
            >
              {submitting ? t.common.submitting : t.verifyEmail.submitButton}
            </Button>
          </form>
        </>
      )}
    </Card>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense fallback={<AuthCardSkeleton />}>
      <VerifyEmailContent />
    </Suspense>
  );
}
