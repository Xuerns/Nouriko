"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";
import { Alert } from "@/components/ui/Alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PasswordInput } from "@/components/ui/PasswordInput";
import { Separator } from "@/components/ui/separator";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";

export default function RegisterPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading, profile } = useAuth();
  const { language, t } = useI18n();
  const { theme } = useTheme();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [registered, setRegistered] = useState(false);
  const [devToken, setDevToken] = useState<string | null>(null);

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.replace(profile?.onboarding_completed ? "/chat" : "/onboarding");
    }
  }, [isAuthenticated, isLoading, profile, router]);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);

    if (password !== confirmPassword) {
      setError(t.common.passwordMismatch);
      return;
    }

    setSubmitting(true);
    try {
      const res = await apiClient.register({
        email: email.trim(),
        password,
        language,
        theme,
      });
      setRegistered(true);
      setDevToken(res.dev_verification_token ?? null);
    } catch (err) {
      const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
      setError(getFriendlyErrorMessage(code, language));
    } finally {
      setSubmitting(false);
    }
  }

  if (isLoading || isAuthenticated) {
    return (
      <Card className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 text-sm text-muted-foreground shadow-none ring-0">
        {t.common.loading}
      </Card>
    );
  }

  if (registered) {
    return (
      <Card className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5 sm:p-7">
        <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground">
          {t.register.successTitle}
        </h1>
        <Alert variant="success">{t.register.successMessage}</Alert>

        {devToken ? (
          <div className="rounded-2xl bg-muted p-4 space-y-2.5 text-sm">
            <p className="text-xs font-medium text-muted-foreground">
              {t.register.devTokenLabel}
            </p>
            <Button
              render={
                <Link
                  href={`/verify-email?token=${encodeURIComponent(devToken)}`}
                />
              }
              size="sm"
              className="h-10 rounded-full px-5 text-xs shadow-none"
            >
              {t.register.devVerifyLink}
            </Button>
          </div>
        ) : (
          <div>
            <Button
              render={<Link href="/verify-email" />}
              variant="outline"
              className="h-10 rounded-full px-5 text-sm shadow-none"
            >
              {t.register.goToVerify}
            </Button>
          </div>
        )}

        <p className="text-sm text-muted-foreground">
          {t.register.hasAccount}{" "}
          <Link
            href="/login"
            className="font-medium text-foreground underline underline-offset-4 transition-colors duration-150 hover:text-primary"
          >
            {t.register.loginLink}
          </Link>
        </p>
      </Card>
    );
  }

  return (
    <Card className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-6 sm:p-7">
      <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground">
        {t.register.title}
      </h1>

      {error ? <Alert variant="error">{error}</Alert> : null}

      <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
        <div className="space-y-1.5">
          <Label
            htmlFor="register-email"
            className="text-sm font-medium text-foreground"
          >
            {t.common.emailLabel}
          </Label>
          <Input
            id="register-email"
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

        <PasswordInput
          id="register-password"
          name="password"
          label={t.common.passwordLabel}
          value={password}
          onChange={setPassword}
          autoComplete="new-password"
          minLength={8}
          disabled={submitting}
          showStrength
          hint={t.common.passwordMinHint}
        />

        <PasswordInput
          id="register-confirm-password"
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
          {submitting ? t.common.submitting : t.register.submitButton}
        </Button>
      </form>

      <Separator className="bg-border" />

      <div className="space-y-2 text-xs text-muted-foreground">
        <p>
          {t.register.privacyPrefix}{" "}
          <Link
            href="/privacy"
            className="font-medium text-foreground underline underline-offset-4 transition-colors duration-150 hover:text-primary"
          >
            {t.common.privacyPolicy}
          </Link>
          .
        </p>
        <p className="text-sm">
          {t.register.hasAccount}{" "}
          <Link
            href="/login"
            className="font-medium text-foreground underline underline-offset-4 transition-colors duration-150 hover:text-primary"
          >
            {t.register.loginLink}
          </Link>
        </p>
      </div>
    </Card>
  );
}
