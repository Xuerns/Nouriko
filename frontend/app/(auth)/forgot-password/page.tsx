"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
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
    <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-6">
      <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
        {t.forgotPassword.title}
      </h1>

      {error ? <Alert variant="error">{error}</Alert> : null}

      {submitted ? (
        <div className="space-y-4">
          <Alert variant="success">{t.forgotPassword.successMessage}</Alert>
          {devResetToken ? (
            <div>
              <Link
                href={`/reset-password?token=${encodeURIComponent(devResetToken)}`}
                className="inline-block rounded-md bg-neutral-900 px-3.5 py-2 text-xs font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200"
              >
                {t.forgotPassword.devResetLink}
              </Link>
            </div>
          ) : null}
        </div>
      ) : (
        <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
          <div className="space-y-1.5">
            <label
              htmlFor="forgot-email"
              className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
            >
              {t.common.emailLabel}
            </label>
            <input
              id="forgot-email"
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

          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200 disabled:opacity-60"
          >
            {submitting ? t.common.submitting : t.forgotPassword.submitButton}
          </button>
        </form>
      )}

      <div className="border-t border-neutral-300 dark:border-neutral-700 pt-4 text-sm">
        <Link
          href="/login"
          className="font-medium text-neutral-900 dark:text-neutral-100 underline underline-offset-4"
        >
          {t.forgotPassword.backToLogin}
        </Link>
      </div>
    </div>
  );
}
