"use client";

import { useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";

export default function ChatShellPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading, profile } = useAuth();
  const { t } = useI18n();

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated) {
      router.replace("/login");
      return;
    }
    if (profile && !profile.onboarding_completed) {
      router.replace("/onboarding");
    }
  }, [isAuthenticated, isLoading, profile, router]);

  if (isLoading || !isAuthenticated || !profile || !profile.onboarding_completed) {
    return (
      <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
        {t.common.loading}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-5 flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
            {t.chat.title}
          </h1>
          <p className="text-sm text-neutral-600 dark:text-neutral-400">
            {t.chat.targetSummary}:{" "}
            <span className="font-mono font-medium text-neutral-900 dark:text-neutral-100">
              {profile.daily_calorie_target ?? "-"} {t.profile.kcalUnit}
            </span>
          </p>
        </div>

        <Link
          href="/profile"
          className="rounded-md border border-neutral-300 dark:border-neutral-700 px-3.5 py-2 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
        >
          {t.chat.editProfileLink}
        </Link>
      </section>

      <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 min-h-72 flex flex-col justify-between gap-6">
        <div className="space-y-3">
          <p className="text-sm text-neutral-600 dark:text-neutral-400">
            {t.chat.subtitle}
          </p>
          <Alert variant="info">{t.chat.shellNotice}</Alert>
        </div>

        <div className="flex flex-col gap-2 border-t border-neutral-300 dark:border-neutral-700 pt-4 sm:flex-row">
          <input
            type="text"
            disabled
            aria-label={t.chat.inputPlaceholder}
            placeholder={t.chat.inputPlaceholder}
            className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500 disabled:opacity-60"
          />
          <button
            type="button"
            disabled
            className="shrink-0 rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white dark:bg-neutral-100 dark:text-neutral-950 disabled:opacity-50"
          >
            {t.chat.sendButton}
          </button>
        </div>
      </section>
    </div>
  );
}
