"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { ChevronLeft, ChevronRight, RotateCcw } from "lucide-react";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { formatIDR, formatMacro } from "@/components/chat/format";
import {
  getHealthLabelTextClass,
  SavedMealItem,
} from "@/components/meals/SavedMealItem";
import { apiClient, ApiError } from "@/lib/api";
import {
  formatDisplayDate,
  getLocalDateString,
  isValidDateString,
  shiftDateString,
} from "@/lib/date";
import { getFriendlyErrorMessage } from "@/lib/errors";
import type { DailySummaryResponse } from "@/lib/types";

function WireframeProgressBar({
  label,
  percent,
}: {
  label: string;
  percent: number;
}) {
  const safePercent = Number.isFinite(percent)
    ? Math.max(0, Math.round(percent))
    : 0;
  const fillWidth = Math.min(100, safePercent);

  return (
    <div
      role="progressbar"
      aria-label={label}
      aria-valuenow={safePercent}
      aria-valuemin={0}
      aria-valuemax={100}
      className="h-2 w-full rounded-sm border border-neutral-300 dark:border-neutral-700 overflow-hidden"
    >
      <div
        className="h-full bg-neutral-900 dark:bg-neutral-100"
        style={{ width: `${fillWidth}%` }}
      />
    </div>
  );
}

function SummaryContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { token, isAuthenticated, isLoading, profile, handleUnauthorized } =
    useAuth();
  const { t, language } = useI18n();

  const todayStr = getLocalDateString();
  const queryDate = searchParams.get("date");
  const selectedDate = isValidDateString(queryDate) ? queryDate : todayStr;

  const [summary, setSummary] = useState<DailySummaryResponse | null>(null);
  const [loadedKey, setLoadedKey] = useState<string | null>(null);
  const [reloadNonce, setReloadNonce] = useState(0);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deletingIds, setDeletingIds] = useState<Record<string, boolean>>({});

  const currentKey = `${selectedDate}:${reloadNonce}`;

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

  useEffect(() => {
    if (
      isLoading ||
      !isAuthenticated ||
      !token ||
      !profile ||
      !profile.onboarding_completed
    ) {
      return;
    }

    let cancelled = false;

    async function run() {
      try {
        const data = await apiClient.getDailySummary(token as string, selectedDate);
        if (cancelled) return;
        setSummary(data);
        setFetchError(null);
        setLoadedKey(currentKey);
      } catch (err) {
        if (cancelled) return;
        if (handleUnauthorized(err)) return;
        if (err instanceof ApiError && err.code === "ONBOARDING_REQUIRED") {
          router.replace("/onboarding");
          return;
        }
        const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
        setFetchError(getFriendlyErrorMessage(code, language));
        setLoadedKey(currentKey);
      }
    }

    void run();
    return () => {
      cancelled = true;
    };
  }, [
    currentKey,
    handleUnauthorized,
    isAuthenticated,
    isLoading,
    language,
    profile,
    router,
    selectedDate,
    token,
  ]);

  const isFetching = loadedKey !== currentKey;

  const triggerReload = useCallback(() => {
    setFetchError(null);
    setReloadNonce((prev) => prev + 1);
  }, []);

  const updateDate = (nextDate: string) => {
    if (!isValidDateString(nextDate)) return;
    setDeleteError(null);
    setFetchError(null);
    router.replace(`/summary?date=${encodeURIComponent(nextDate)}`, {
      scroll: false,
    });
  };

  const handleDeleteMeal = async (mealId: string) => {
    if (!token || deletingIds[mealId]) return;

    setDeletingIds((prev) => ({ ...prev, [mealId]: true }));
    setDeleteError(null);

    try {
      await apiClient.deleteMeal(token, mealId);
      setReloadNonce((prev) => prev + 1);
    } catch (err) {
      if (handleUnauthorized(err)) return;
      if (err instanceof ApiError && err.code === "ONBOARDING_REQUIRED") {
        router.replace("/onboarding");
        return;
      }
      const code =
        err instanceof ApiError && err.code === "TOO_MANY_REQUESTS"
          ? "TOO_MANY_REQUESTS"
          : "MEAL_DELETE_FAILED";
      setDeleteError(getFriendlyErrorMessage(code, language));
    } finally {
      setDeletingIds((prev) => {
        const next = { ...prev };
        delete next[mealId];
        return next;
      });
    }
  };

  if (
    isLoading ||
    !isAuthenticated ||
    !profile ||
    !profile.onboarding_completed
  ) {
    return (
      <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
        {t.common.loading}
      </div>
    );
  }

  const totals = summary?.totals;
  const targets = summary?.targets;
  const progress = summary?.progress_percent;
  const meals = summary?.meals ?? [];

  const dailyHealthLabelText =
    totals?.health_label &&
    Object.hasOwn(t.chat.healthLabels, totals.health_label)
      ? t.chat.healthLabels[totals.health_label]
      : null;

  const macroRows = [
    {
      key: "protein",
      label: t.profile.protein,
      actual: totals?.protein_g ?? 0,
      target: targets?.target_protein_g ?? 0,
      percent: progress?.protein_g ?? 0,
    },
    {
      key: "carbs",
      label: t.profile.carbs,
      actual: totals?.carbs_g ?? 0,
      target: targets?.target_carbs_g ?? 0,
      percent: progress?.carbs_g ?? 0,
    },
    {
      key: "fat",
      label: t.profile.fat,
      actual: totals?.fat_g ?? 0,
      target: targets?.target_fat_g ?? 0,
      percent: progress?.fat_g ?? 0,
    },
    {
      key: "fiber",
      label: t.profile.fiber,
      actual: totals?.fiber_g ?? 0,
      target: targets?.target_fiber_g ?? 0,
      percent: progress?.fiber_g ?? 0,
    },
  ];

  return (
    <div className="space-y-5">
      <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 sm:p-5 flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
            {t.summary.title}
          </h1>
          <p className="text-sm text-neutral-600 dark:text-neutral-400">
            {formatDisplayDate(selectedDate, language)}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => updateDate(shiftDateString(selectedDate, -1))}
            aria-label={t.summary.prevDayLabel}
            className="inline-flex items-center justify-center rounded-md border border-neutral-300 dark:border-neutral-700 p-2 text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
          >
            <ChevronLeft className="h-4 w-4 shrink-0" aria-hidden="true" />
          </button>

          <input
            type="date"
            value={selectedDate}
            onChange={(e) => {
              if (e.target.value) {
                updateDate(e.target.value);
              }
            }}
            aria-label={t.summary.dateInputLabel}
            className="rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-2.5 py-1.5 font-mono text-xs text-neutral-900 dark:text-neutral-100"
          />

          <button
            type="button"
            onClick={() => updateDate(shiftDateString(selectedDate, 1))}
            aria-label={t.summary.nextDayLabel}
            className="inline-flex items-center justify-center rounded-md border border-neutral-300 dark:border-neutral-700 p-2 text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
          >
            <ChevronRight className="h-4 w-4 shrink-0" aria-hidden="true" />
          </button>

          <button
            type="button"
            disabled={selectedDate === todayStr}
            onClick={() => updateDate(todayStr)}
            className="rounded-md border border-neutral-300 dark:border-neutral-700 px-3 py-1.5 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
          >
            {t.summary.todayButton}
          </button>
        </div>
      </section>

      {fetchError ? (
        <div className="space-y-3">
          <Alert variant="error">{fetchError}</Alert>
          <button
            type="button"
            onClick={triggerReload}
            className="inline-flex items-center gap-1.5 rounded-md border border-neutral-300 dark:border-neutral-700 px-3 py-1.5 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
          >
            <RotateCcw className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            <span>{t.summary.retryButton}</span>
          </button>
        </div>
      ) : isFetching && !summary ? (
        <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
          {t.common.loading}
        </div>
      ) : (
        <>
          <section className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 space-y-2.5">
              <p className="font-mono text-xs uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
                {t.summary.caloriesHeading}
              </p>
              <div className="flex items-baseline justify-between gap-2">
                <p className="font-mono text-lg font-semibold text-neutral-900 dark:text-neutral-100">
                  {totals?.calories ?? 0} {t.profile.kcalUnit}
                </p>
                <span className="font-mono text-xs text-neutral-600 dark:text-neutral-400">
                  {progress?.calories ?? 0}%
                </span>
              </div>
              <p className="text-xs text-neutral-600 dark:text-neutral-400">
                {t.summary.ofTarget}{" "}
                <span className="font-mono text-neutral-900 dark:text-neutral-100">
                  {targets?.daily_calorie_target ?? 0} {t.profile.kcalUnit}
                </span>
              </p>
              <WireframeProgressBar
                label={t.summary.caloriesHeading}
                percent={progress?.calories ?? 0}
              />
            </div>

            <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 space-y-2.5">
              <p className="font-mono text-xs uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
                {t.summary.spendingHeading}
              </p>
              <p className="font-mono text-lg font-semibold text-neutral-900 dark:text-neutral-100">
                {formatIDR(totals?.total_spending_idr ?? 0)}
              </p>
              <p className="text-xs text-neutral-600 dark:text-neutral-400">
                {(totals?.meal_count ?? 0) > 0
                  ? `${totals?.meal_count ?? 0} ${t.meals.mealCountUnit}`
                  : t.summary.noMealsCount}
              </p>
            </div>

            <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 space-y-2.5">
              <p className="font-mono text-xs uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
                {t.summary.healthScoreHeading}
              </p>
              {(totals?.meal_count ?? 0) > 0 && totals?.health_label ? (
                <div className="space-y-1">
                  <p className="font-mono text-lg font-semibold text-neutral-900 dark:text-neutral-100">
                    {totals.average_health_score}/100
                  </p>
                  <p
                    className={`text-sm font-medium ${getHealthLabelTextClass(totals.health_label)}`}
                  >
                    {dailyHealthLabelText}
                  </p>
                </div>
              ) : (
                <p className="font-mono text-lg font-semibold text-neutral-500 dark:text-neutral-400">
                  -
                </p>
              )}
            </div>
          </section>

          <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 sm:p-5 space-y-3">
            <h2 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
              {t.summary.macrosHeading}
            </h2>

            <div className="grid grid-cols-2 sm:grid-cols-4 rounded-md border border-neutral-300 dark:border-neutral-700 divide-y sm:divide-y-0 sm:divide-x divide-neutral-300 dark:divide-neutral-700">
              {macroRows.map((macro) => (
                <div key={macro.key} className="p-3 space-y-2">
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-xs font-medium text-neutral-600 dark:text-neutral-400">
                      {macro.label}
                    </p>
                    <span className="font-mono text-xs text-neutral-600 dark:text-neutral-400">
                      {macro.percent}%
                    </span>
                  </div>
                  <p className="font-mono text-sm font-semibold text-neutral-900 dark:text-neutral-100">
                    {formatMacro(macro.actual)} / {formatMacro(macro.target)}{" "}
                    {t.profile.gramUnit}
                  </p>
                  <WireframeProgressBar
                    label={macro.label}
                    percent={macro.percent}
                  />
                </div>
              ))}
            </div>
          </section>

          <section className="space-y-3">
            <div className="flex items-center justify-between gap-2">
              <h2 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
                {t.summary.savedMealsHeading}
              </h2>
              <span className="font-mono text-xs text-neutral-600 dark:text-neutral-400">
                {meals.length} {t.meals.mealCountUnit}
              </span>
            </div>

            {deleteError ? <Alert variant="error">{deleteError}</Alert> : null}

            {meals.length === 0 ? (
              <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-3">
                <p className="text-sm font-medium text-neutral-900 dark:text-neutral-100">
                  {t.summary.emptyTitle}
                </p>
                <p className="text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed">
                  {t.summary.emptyDescription}
                </p>
                <div>
                  <Link
                    href="/chat"
                    className="inline-flex items-center rounded-md border border-neutral-900 dark:border-neutral-100 px-3.5 py-2 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
                  >
                    {t.summary.goToChatButton}
                  </Link>
                </div>
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                {meals.map((meal) => (
                  <SavedMealItem
                    key={meal.id}
                    meal={meal}
                    onDelete={handleDeleteMeal}
                    isDeleting={Boolean(deletingIds[meal.id])}
                  />
                ))}
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
}

export default function SummaryPage() {
  return (
    <Suspense
      fallback={
        <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
          Loading...
        </div>
      }
    >
      <SummaryContent />
    </Suspense>
  );
}
