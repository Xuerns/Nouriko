"use client";

import { Suspense, useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { PasswordInput } from "@/components/ui/PasswordInput";
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
      <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
        {t.common.loading}
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-6">
      <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
        {t.login.title}
      </h1>

      {sessionExpired && !error ? (
        <Alert variant="info">{t.login.sessionExpired}</Alert>
      ) : null}

      {error ? <Alert variant="error">{error}</Alert> : null}

      <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
        <div className="space-y-1.5">
          <label
            htmlFor="login-email"
            className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
          >
            {t.common.emailLabel}
          </label>
          <input
            id="login-email"
            name="email"
            type="email"
            autoComplete="email"
            required
            disabled={submitting}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={t.common.emailPlaceholder}
            className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
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
            className="text-xs font-medium text-neutral-700 dark:text-neutral-300 underline underline-offset-4 hover:text-neutral-900 dark:hover:text-neutral-100"
          >
            {t.login.forgotPasswordLink}
          </Link>
        </div>

        <button
          type="submit"
          disabled={submitting}
          className="w-full rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200 disabled:opacity-60"
        >
          {submitting ? t.common.submitting : t.login.submitButton}
        </button>
      </form>

      <div className="space-y-2 border-t border-neutral-300 dark:border-neutral-700 pt-4 text-sm text-neutral-600 dark:text-neutral-400">
        <p>
          {t.login.noAccount}{" "}
          <Link
            href="/register"
            className="font-medium text-neutral-900 dark:text-neutral-100 underline underline-offset-4"
          >
            {t.login.registerLink}
          </Link>
        </p>
        <p className="text-xs">
          <Link
            href="/privacy"
            className="underline underline-offset-4 hover:text-neutral-900 dark:hover:text-neutral-100"
          >
            {t.common.privacyPolicy}
          </Link>
        </p>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
          ...
        </div>
      }
    >
      <LoginContent />
    </Suspense>
  );
}
