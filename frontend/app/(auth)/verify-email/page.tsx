"use client";

import { Suspense, useEffect, useRef, useState, type FormEvent } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";

function VerifyEmailContent() {
  const searchParams = useSearchParams();
  const { language, t } = useI18n();
  const initialToken = searchParams.get("token") ?? "";

  const [token, setToken] = useState(initialToken);
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
    <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-5">
      <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
        {t.verifyEmail.title}
      </h1>

      {verified ? (
        <div className="space-y-4">
          <Alert variant="success">{t.verifyEmail.successMessage}</Alert>
          <Link
            href="/login"
            className="inline-block rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200"
          >
            {t.verifyEmail.goToLogin}
          </Link>
        </div>
      ) : (
        <>
          {error ? <Alert variant="error">{error}</Alert> : null}
          {submitting && token ? (
            <p className="text-sm text-neutral-600 dark:text-neutral-400">
              {t.verifyEmail.verifying}
            </p>
          ) : null}

          <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
            <div className="space-y-1.5">
              <label
                htmlFor="verify-token"
                className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
              >
                {t.verifyEmail.tokenLabel}
              </label>
              <input
                id="verify-token"
                name="token"
                type="text"
                required
                disabled={submitting}
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder={t.verifyEmail.tokenPlaceholder}
                className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 font-mono text-sm text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
              />
            </div>

            <button
              type="submit"
              disabled={submitting}
              className="w-full rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200 disabled:opacity-60"
            >
              {submitting ? t.common.submitting : t.verifyEmail.submitButton}
            </button>
          </form>
        </>
      )}
    </div>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
          ...
        </div>
      }
    >
      <VerifyEmailContent />
    </Suspense>
  );
}
