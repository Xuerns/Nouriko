"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ChevronDown, ChevronUp, RotateCcw } from "lucide-react";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { formatIDR } from "@/components/chat/format";
import {
  getHealthLabelTextClass,
  SavedMealItem,
} from "@/components/meals/SavedMealItem";
import { WeeklyHealthChart } from "@/components/meals/WeeklyHealthChart";
import { apiClient, ApiError } from "@/lib/api";
import { formatDisplayDate, getLocalDateString } from "@/lib/date";
import { getFriendlyErrorMessage } from "@/lib/errors";
import type { MealHistoryResponse } from "@/lib/types";

type HistoryRange = 7 | 14 | 30;

export default function HistoryPage() {
  const router = useRouter();
  const { token, isAuthenticated, isLoading, profile, handleUnauthorized } =
    useAuth();
  const { t, language } = useI18n();

  const [daysRange, setDaysRange] = useState<HistoryRange>(30);
  const [history, setHistory] = useState<MealHistoryResponse | null>(null);
  const [loadedKey, setLoadedKey] = useState<string | null>(null);
  const [reloadNonce, setReloadNonce] = useState(0);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [expandedDates, setExpandedDates] = useState<Record<string, boolean>>(
    {}
  );

  const currentKey = `${daysRange}:${reloadNonce}`;

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
        const data = await apiClient.getMealHistory(token as string, {
          to: getLocalDateString(),
          days: daysRange,
        });
        if (cancelled) return;
        setHistory(data);
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
    daysRange,
    handleUnauthorized,
    isAuthenticated,
    isLoading,
    language,
    profile,
    router,
    token,
  ]);

  const isFetching = loadedKey !== currentKey;

  const triggerReload = useCallback(() => {
    setFetchError(null);
    setReloadNonce((prev) => prev + 1);
  }, []);

  const toggleDate = (dateStr: string) => {
    setExpandedDates((prev) => ({ ...prev, [dateStr]: !prev[dateStr] }));
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

  const rangeOptions: { value: HistoryRange; label: string }[] = [
    { value: 7, label: t.history.range7Days },
    { value: 14, label: t.history.range14Days },
    { value: 30, label: t.history.range30Days },
  ];

  const days = history?.days ?? [];
  const weeklyTrend = history?.weekly_trend ?? [];

  return (
    <div className="space-y-5">
      <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 sm:p-5 flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
            {t.history.title}
          </h1>
          <p className="text-sm text-neutral-600 dark:text-neutral-400">
            {t.history.subtitle}
          </p>
        </div>

        <div
          role="group"
          aria-label={t.history.rangeLabel}
          className="flex flex-wrap items-center gap-2"
        >
          {rangeOptions.map((option) => {
            const active = daysRange === option.value;
            return (
              <button
                key={option.value}
                type="button"
                aria-pressed={active}
                onClick={() => setDaysRange(option.value)}
                className={`rounded-md border px-3 py-1.5 text-xs font-medium transition-colors duration-150 ${
                  active
                    ? "border-neutral-900 dark:border-neutral-100 text-neutral-900 dark:text-neutral-100"
                    : "border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-900"
                }`}
              >
                {option.label}
              </button>
            );
          })}
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
            <span>{t.history.retryButton}</span>
          </button>
        </div>
      ) : isFetching && !history ? (
        <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
          {t.common.loading}
        </div>
      ) : (
        <>
          {weeklyTrend.length > 0 ? (
            <WeeklyHealthChart points={weeklyTrend} />
          ) : null}

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
              {t.history.dailyListHeading}
            </h2>

            {days.length === 0 ? (
              <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-3">
                <p className="text-sm font-medium text-neutral-900 dark:text-neutral-100">
                  {t.history.emptyTitle}
                </p>
                <p className="text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed">
                  {t.history.emptyDescription}
                </p>
                <div>
                  <Link
                    href="/chat"
                    className="inline-flex items-center rounded-md border border-neutral-900 dark:border-neutral-100 px-3.5 py-2 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
                  >
                    {t.history.goToChatButton}
                  </Link>
                </div>
              </div>
            ) : (
              <div className="space-y-3">
                {days.map((day) => {
                  const isExpanded = Boolean(expandedDates[day.date]);
                  const dayHealthLabel = Object.hasOwn(
                    t.chat.healthLabels,
                    day.health_label
                  )
                    ? t.chat.healthLabels[day.health_label]
                    : t.chat.healthLabels.fair;

                  return (
                    <article
                      key={day.date}
                      className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 space-y-3"
                    >
                      <div className="flex flex-wrap items-start justify-between gap-3">
                        <div className="space-y-0.5">
                          <h3 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
                            {formatDisplayDate(day.date, language)}
                          </h3>
                          <p className="font-mono text-xs text-neutral-500 dark:text-neutral-400">
                            {day.date} — {day.meal_count}{" "}
                            {t.meals.mealCountUnit}
                          </p>
                        </div>

                        <div className="flex flex-wrap items-center gap-2">
                          <button
                            type="button"
                            onClick={() => toggleDate(day.date)}
                            aria-expanded={isExpanded}
                            className="inline-flex items-center gap-1.5 rounded-md border border-neutral-300 dark:border-neutral-700 px-3 py-1.5 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
                          >
                            {isExpanded ? (
                              <ChevronUp
                                className="h-3.5 w-3.5 shrink-0"
                                aria-hidden="true"
                              />
                            ) : (
                              <ChevronDown
                                className="h-3.5 w-3.5 shrink-0"
                                aria-hidden="true"
                              />
                            )}
                            <span>
                              {isExpanded
                                ? t.history.collapseMealsButton
                                : t.history.expandMealsButton}
                            </span>
                          </button>

                          <Link
                            href={`/summary?date=${encodeURIComponent(day.date)}`}
                            className="rounded-md border border-neutral-300 dark:border-neutral-700 px-3 py-1.5 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
                          >
                            {t.history.viewSummaryLink}
                          </Link>
                        </div>
                      </div>

                      <div className="grid grid-cols-1 sm:grid-cols-3 rounded-md border border-neutral-300 dark:border-neutral-700 divide-y sm:divide-y-0 sm:divide-x divide-neutral-300 dark:divide-neutral-700">
                        <div className="p-2.5">
                          <p className="text-xs text-neutral-500 dark:text-neutral-400">
                            {t.history.totalCaloriesLabel}
                          </p>
                          <p className="mt-0.5 font-mono text-sm font-medium text-neutral-900 dark:text-neutral-100">
                            {day.total_calories} {t.profile.kcalUnit}
                          </p>
                        </div>

                        <div className="p-2.5">
                          <p className="text-xs text-neutral-500 dark:text-neutral-400">
                            {t.history.totalSpendingLabel}
                          </p>
                          <p className="mt-0.5 font-mono text-sm font-medium text-neutral-900 dark:text-neutral-100">
                            {formatIDR(day.total_spending_idr)}
                          </p>
                        </div>

                        <div className="p-2.5">
                          <p className="text-xs text-neutral-500 dark:text-neutral-400">
                            {t.history.averageScoreLabel}
                          </p>
                          <p className="mt-0.5 font-mono text-sm font-medium text-neutral-900 dark:text-neutral-100">
                            {day.average_health_score}/100{" "}
                            <span
                              className={`font-sans ${getHealthLabelTextClass(day.health_label)}`}
                            >
                              ({dayHealthLabel})
                            </span>
                          </p>
                        </div>
                      </div>

                      {isExpanded ? (
                        <div className="grid grid-cols-1 gap-3 border-t border-neutral-300 dark:border-neutral-700 pt-3 sm:grid-cols-2">
                          {day.meals.map((meal) => (
                            <SavedMealItem key={meal.id} meal={meal} />
                          ))}
                        </div>
                      ) : null}
                    </article>
                  );
                })}
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
}
