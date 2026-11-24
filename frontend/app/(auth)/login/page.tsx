"use client";

import { Suspense, useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PasswordInput } from "@/components/ui/PasswordInput";
import { Separator } from "@/components/ui/separator";
import { ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";

function LoginContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { isAuthenticated, isLoading, login, profile } = useAuth();
  const { language, t } = useI18n();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const sessionExpired = searchParams.get("reason") === "session_expired";

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.replace(profile?.onboarding_completed ? "/chat" : "/onboarding");
    }
  }, [isAuthenticated, isLoading, profile, router]);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      const loggedInProfile = await login(email.trim(), password);
      router.replace(
        loggedInProfile.onboarding_completed ? "/chat" : "/onboarding"
      );
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

  return (
    <Card className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-6 sm:p-7">
      <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground">
        {t.login.title}
      </h1>

      {sessionExpired && !error ? (
        <Alert variant="info">{t.login.sessionExpired}</Alert>
      ) : null}

      {error ? <Alert variant="error">{error}</Alert> : null}

      <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
        <div className="space-y-1.5">
          <Label
            htmlFor="login-email"
            className="text-sm font-medium text-foreground"
          >
            {t.common.emailLabel}
          </Label>
          <Input
            id="login-email"
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
          id="login-password"
          name="password"
          label={t.common.passwordLabel}
          value={password}
          onChange={setPassword}
          autoComplete="current-password"
          disabled={submitting}
        />

        <div className="text-right">
          <Link
            href="/forgot-password"
            className="text-xs font-medium text-muted-foreground underline underline-offset-4 transition-colors duration-150 hover:text-primary"
          >
            {t.login.forgotPasswordLink}
          </Link>
        </div>

        <Button
          type="submit"
          disabled={submitting}
          className="h-11 w-full rounded-full text-sm font-medium shadow-none"
        >
          {submitting ? t.common.submitting : t.login.submitButton}
        </Button>
      </form>

      <Separator className="bg-border" />

      <div className="space-y-2 text-sm text-muted-foreground">
        <p>
          {t.login.noAccount}{" "}
          <Link
            href="/register"
            className="font-medium text-foreground underline underline-offset-4 transition-colors duration-150 hover:text-primary"
          >
            {t.login.registerLink}
          </Link>
        </p>
        <p className="text-xs">
          <Link
            href="/privacy"
            className="underline underline-offset-4 transition-colors duration-150 hover:text-primary"
          >
            {t.common.privacyPolicy}
          </Link>
        </p>
      </div>
    </Card>
  );
}

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <Card className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 text-sm text-muted-foreground shadow-none ring-0">
          ...
        </Card>
      }
    >
      <LoginContent />
    </Suspense>
  );
}
