"use client";

import { Suspense, useEffect, useState, type FormEvent } from "react";
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
import { PasswordInput } from "@/components/ui/PasswordInput";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";
import { focusFieldById } from "@/lib/form-validation";
import { cn } from "@/lib/utils";

function ResetPasswordContent() {
  const searchParams = useSearchParams();
  const { language, t } = useI18n();
  const initialToken = searchParams.get("token") ?? "";

  const [token, setToken] = useState(initialToken);
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [tokenError, setTokenError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [confirmError, setConfirmError] = useState<string | null>(null);
  const [dirty, setDirty] = useState({
    token: false,
    password: false,
    confirm: false,
  });
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (initialToken) {
      window.history.replaceState(null, "", window.location.pathname);
    }
  }, [initialToken]);

  function validateToken(value: string): string | null {
    return value.trim() ? null : t.common.tokenRequired;
  }

  function validatePassword(value: string): string | null {
    if (!value) return t.common.passwordRequired;
    if (value.length < 8) return t.common.passwordMinHint;
    return null;
  }

  function validateConfirm(value: string, currentPassword: string): string | null {
    if (!value) return t.common.confirmPasswordRequired;
    if (value !== currentPassword) return t.common.passwordMismatch;
    return null;
  }

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);

    const nextTokenErr = validateToken(token);
    const nextPassErr = validatePassword(newPassword);
    const nextConfirmErr = validateConfirm(confirmPassword, newPassword);

    setTokenError(nextTokenErr);
    setPasswordError(nextPassErr);
    setConfirmError(nextConfirmErr);

    if (nextTokenErr) {
      focusFieldById("reset-token");
      return;
    }
    if (nextPassErr) {
      focusFieldById("reset-new-password");
      return;
    }
    if (nextConfirmErr) {
      focusFieldById("reset-confirm-password");
      return;
    }

    setSubmitting(true);
    try {
      await apiClient.resetPassword({
        token: token.trim(),
        new_password: newPassword,
      });
      setSuccess(true);
    } catch (err) {
      const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
      setError(getFriendlyErrorMessage(code, language));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Card className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-6 sm:p-7">
      <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground">
        {t.resetPassword.title}
      </h1>

      {success ? (
        <div className="space-y-4">
          <Alert variant="success">{t.resetPassword.successMessage}</Alert>
          <Button
            render={<Link href="/login" />}
            nativeButton={false}
            className="h-11 rounded-full px-5 text-sm shadow-none"
          >
            {t.resetPassword.goToLogin}
          </Button>
        </div>
      ) : (
        <>
          {error ? <Alert variant="error">{error}</Alert> : null}

          <form
            noValidate
            onSubmit={(e) => void handleSubmit(e)}
            className="space-y-4"
          >
            <div className="space-y-1.5">
              <Label
                htmlFor="reset-token"
                className="text-sm font-medium text-foreground"
              >
                {t.resetPassword.tokenLabel}
              </Label>
              <Input
                id="reset-token"
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
                  setDirty((prev) => ({ ...prev, token: true }));
                  if (tokenError && !validateToken(next)) {
                    setTokenError(null);
                  }
                }}
                onBlur={() => {
                  if (dirty.token) {
                    setTokenError(validateToken(token));
                  }
                }}
                aria-invalid={Boolean(tokenError)}
                aria-describedby={tokenError ? "reset-token-error" : undefined}
                placeholder={t.resetPassword.tokenPlaceholder}
                className={cn(
                  "h-11 rounded-2xl bg-card px-3.5 font-mono text-sm shadow-none",
                  tokenError &&
                    "border-[var(--error-text)] focus-visible:border-[var(--error-text)]"
                )}
              />
              <FieldError id="reset-token-error" message={tokenError} />
            </div>

            <PasswordInput
              id="reset-new-password"
              name="newPassword"
              label={t.common.newPasswordLabel}
              value={newPassword}
              onChange={(next) => {
                setNewPassword(next);
                setDirty((prev) => ({ ...prev, password: true }));
                if (passwordError && !validatePassword(next)) {
                  setPasswordError(null);
                }
                if (confirmError && !validateConfirm(confirmPassword, next)) {
                  setConfirmError(null);
                }
              }}
              onBlur={() => {
                if (dirty.password) {
                  setPasswordError(validatePassword(newPassword));
                }
              }}
              autoComplete="new-password"
              minLength={8}
              disabled={submitting}
              showStrength
              hint={t.common.passwordMinHint}
              error={passwordError}
            />

            <PasswordInput
              id="reset-confirm-password"
              name="confirmPassword"
              label={t.common.confirmPasswordLabel}
              value={confirmPassword}
              onChange={(next) => {
                setConfirmPassword(next);
                setDirty((prev) => ({ ...prev, confirm: true }));
                if (confirmError && !validateConfirm(next, newPassword)) {
                  setConfirmError(null);
                }
              }}
              onBlur={() => {
                if (dirty.confirm) {
                  setConfirmError(validateConfirm(confirmPassword, newPassword));
                }
              }}
              autoComplete="new-password"
              minLength={8}
              disabled={submitting}
              error={confirmError}
            />

            <Button
              type="submit"
              disabled={submitting}
              className="h-11 w-full rounded-full text-sm font-medium shadow-none"
            >
              {submitting ? t.common.submitting : t.resetPassword.submitButton}
            </Button>
          </form>
        </>
      )}
    </Card>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={<AuthCardSkeleton />}>
      <ResetPasswordContent />
    </Suspense>
  );
}
