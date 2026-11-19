"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";
import { Alert } from "@/components/ui/Alert";
import { PasswordInput } from "@/components/ui/PasswordInput";
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
      <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
        {t.common.loading}
      </div>
    );
  }

  if (registered) {
    return (
      <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-5">
        <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
          {t.register.successTitle}
        </h1>
        <Alert variant="success">{t.register.successMessage}</Alert>

        {devToken ? (
          <div className="rounded-md border border-neutral-300 dark:border-neutral-700 p-4 space-y-2 text-sm">
            <p className="text-xs font-medium text-neutral-600 dark:text-neutral-400">
              {t.register.devTokenLabel}
            </p>
            <Link
              href={`/verify-email?token=${encodeURIComponent(devToken)}`}
              className="inline-block rounded-md bg-neutral-900 px-3.5 py-2 text-xs font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200"
            >
              {t.register.devVerifyLink}
            </Link>
          </div>
        ) : (
          <div>
            <Link
              href="/verify-email"
              className="inline-block rounded-md border border-neutral-300 dark:border-neutral-700 px-4 py-2 text-sm font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
            >
              {t.register.goToVerify}
            </Link>
          </div>
        )}

        <p className="text-sm text-neutral-600 dark:text-neutral-400">
          {t.register.hasAccount}{" "}
          <Link
            href="/login"
            className="font-medium text-neutral-900 dark:text-neutral-100 underline underline-offset-4"
          >
            {t.register.loginLink}
          </Link>
        </p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-6">
      <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
        {t.register.title}
      </h1>

      {error ? <Alert variant="error">{error}</Alert> : null}

      <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
        <div className="space-y-1.5">
          <label
            htmlFor="register-email"
            className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
          >
            {t.common.emailLabel}
          </label>
          <input
            id="register-email"
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

        <button
          type="submit"
          disabled={submitting}
          className="w-full rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200 disabled:opacity-60"
        >
          {submitting ? t.common.submitting : t.register.submitButton}
        </button>
      </form>

      <div className="space-y-2 border-t border-neutral-300 dark:border-neutral-700 pt-4 text-xs text-neutral-600 dark:text-neutral-400">
        <p>
          {t.register.privacyPrefix}{" "}
          <Link
            href="/privacy"
            className="font-medium text-neutral-900 dark:text-neutral-100 underline underline-offset-4"
          >
            {t.common.privacyPolicy}
          </Link>
          .
        </p>
        <p className="text-sm">
          {t.register.hasAccount}{" "}
          <Link
            href="/login"
            className="font-medium text-neutral-900 dark:text-neutral-100 underline underline-offset-4"
          >
            {t.register.loginLink}
          </Link>
        </p>
      </div>
    </div>
  );
}
