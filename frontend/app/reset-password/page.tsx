"use client";

import { Suspense, useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
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
    <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-6">
      <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
        {t.resetPassword.title}
      </h1>

      {success ? (
        <div className="space-y-4">
          <Alert variant="success">{t.resetPassword.successMessage}</Alert>
          <Link
            href="/login"
            className="inline-block rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200"
          >
            {t.resetPassword.goToLogin}
          </Link>
        </div>
      ) : (
        <>
          {error ? <Alert variant="error">{error}</Alert> : null}

          <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
            <div className="space-y-1.5">
              <label
                htmlFor="reset-token"
                className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
              >
                {t.resetPassword.tokenLabel}
              </label>
              <input
                id="reset-token"
                name="token"
                type="text"
                required
                disabled={submitting}
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder={t.resetPassword.tokenPlaceholder}
                className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 font-mono text-sm text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
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

            <button
              type="submit"
              disabled={submitting}
              className="w-full rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200 disabled:opacity-60"
            >
              {submitting ? t.common.submitting : t.resetPassword.submitButton}
            </button>
          </form>
        </>
      )}
    </div>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
          ...
        </div>
      }
    >
      <ResetPasswordContent />
    </Suspense>
  );
}
