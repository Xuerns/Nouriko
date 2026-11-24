"use client";

import { Suspense, useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AuthCardSkeleton } from "@/components/ui/PageSkeletons";
import { PasswordInput } from "@/components/ui/PasswordInput";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";

function ResetPasswordContent() {
  const searchParams = useSearchParams();
  const { language, t } = useI18n();
  const initialToken = searchParams.get("token") ?? "";

  const [token, setToken] = useState(initialToken);
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (initialToken) {
      window.history.replaceState(null, "", window.location.pathname);
    }
  }, [initialToken]);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);

    if (newPassword !== confirmPassword) {
      setError(t.common.passwordMismatch);
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
            className="h-11 rounded-full px-5 text-sm shadow-none"
          >
            {t.resetPassword.goToLogin}
          </Button>
        </div>
      ) : (
        <>
          {error ? <Alert variant="error">{error}</Alert> : null}

          <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
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
                required
                disabled={submitting}
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder={t.resetPassword.tokenPlaceholder}
                className="h-11 rounded-2xl bg-card px-3.5 font-mono text-sm shadow-none"
              />
            </div>

            <PasswordInput
              id="reset-new-password"
              name="newPassword"
              label={t.common.newPasswordLabel}
              value={newPassword}
              onChange={setNewPassword}
              autoComplete="new-password"
              minLength={8}
              disabled={submitting}
              showStrength
              hint={t.common.passwordMinHint}
            />

            <PasswordInput
              id="reset-confirm-password"
              name="confirmPassword"
              label={t.common.confirmPasswordLabel}
              value={confirmPassword}
              onChange={setConfirmPassword}
              autoComplete="new-password"
              minLength={8}
              disabled={submitting}
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
