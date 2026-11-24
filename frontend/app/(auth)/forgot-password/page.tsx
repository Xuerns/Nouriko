"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";

export default function ForgotPasswordPage() {
  const { language, t } = useI18n();
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [devResetToken, setDevResetToken] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      const res = await apiClient.forgotPassword(email.trim());
      setSubmitted(true);
      setDevResetToken(res.dev_reset_token ?? null);
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
        {t.forgotPassword.title}
      </h1>

      {error ? <Alert variant="error">{error}</Alert> : null}

      {submitted ? (
        <div className="space-y-4">
          <Alert variant="success">{t.forgotPassword.successMessage}</Alert>
          {devResetToken ? (
            <div>
              <Button
                render={
                  <Link
                    href={`/reset-password?token=${encodeURIComponent(devResetToken)}`}
                  />
                }
                size="sm"
                className="h-10 rounded-full px-5 text-xs shadow-none"
              >
                {t.forgotPassword.devResetLink}
              </Button>
            </div>
          ) : null}
        </div>
      ) : (
        <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
          <div className="space-y-1.5">
            <Label
              htmlFor="forgot-email"
              className="text-sm font-medium text-foreground"
            >
              {t.common.emailLabel}
            </Label>
            <Input
              id="forgot-email"
              name="email"
              type="email"
              autoComplete="email"
              required
              disabled={submitting}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t.common.emailPlaceholder}
              className="h-11 rounded-2xl bg-card px-3.5 text-sm shadow-none"
            />
          </div>

          <Button
            type="submit"
            disabled={submitting}
            className="h-11 w-full rounded-full text-sm font-medium shadow-none"
          >
            {submitting ? t.common.submitting : t.forgotPassword.submitButton}
          </Button>
        </form>
      )}

      <Separator className="bg-border" />

      <div className="text-sm">
        <Link
          href="/login"
          className="font-medium text-foreground underline underline-offset-4 transition-colors duration-150 hover:text-primary"
        >
          {t.forgotPassword.backToLogin}
        </Link>
      </div>
    </Card>
  );
}
