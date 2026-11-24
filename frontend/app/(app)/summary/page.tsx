"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { ChevronLeft, ChevronRight, RotateCcw } from "lucide-react";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { formatIDR, formatMacro } from "@/components/chat/format";
import { getHealthBadgeClass } from "@/components/chat/MealResultCard";
import { SavedMealItem } from "@/components/meals/SavedMealItem";
import { apiClient, ApiError } from "@/lib/api";
import {
  formatDisplayDate,
  getLocalDateString,
  isValidDateString,
  shiftDateString,
} from "@/lib/date";
import { getFriendlyErrorMessage } from "@/lib/errors";
import type { DailySummaryResponse } from "@/lib/types";

function MacroProgressBar({
  label,
  percent,
  colorVar,
}: {
  label: string;
  percent: number;
  colorVar: string;
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
      className="h-2.5 w-full overflow-hidden rounded-full bg-muted"
    >
      <div
        className="h-full rounded-full transition-all duration-200"
        style={{ width: `${fillWidth}%`, backgroundColor: colorVar }}
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
        const data = await apiClient.getDailySummary(
          token as string,
          selectedDate
        );
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
      <Card className="rounded-3xl border border-border bg-card p-6 text-sm text-muted-foreground shadow-none ring-0">
        {t.common.loading}
      </Card>
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
      colorVar: "var(--macro-protein)",
    },
    {
      key: "carbs",
      label: t.profile.carbs,
      actual: totals?.carbs_g ?? 0,
      target: targets?.target_carbs_g ?? 0,
      percent: progress?.carbs_g ?? 0,
      colorVar: "var(--macro-carbs)",
    },
    {
      key: "fat",
      label: t.profile.fat,
      actual: totals?.fat_g ?? 0,
      target: targets?.target_fat_g ?? 0,
      percent: progress?.fat_g ?? 0,
      colorVar: "var(--macro-fat)",
    },
    {
      key: "fiber",
      label: t.profile.fiber,
      actual: totals?.fiber_g ?? 0,
      target: targets?.target_fiber_g ?? 0,
      percent: progress?.fiber_g ?? 0,
      colorVar: "var(--macro-fiber)",
    },
  ];

  return (
    <div className="space-y-5">
      <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="space-y-1">
            <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground sm:text-[26px]">
              {t.summary.title}
            </h1>
            <p className="text-sm text-muted-foreground">
              {formatDisplayDate(selectedDate, language)}
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={() => updateDate(shiftDateString(selectedDate, -1))}
              aria-label={t.summary.prevDayLabel}
              className="size-10 rounded-full shadow-none"
            >
              <ChevronLeft className="size-4 stroke-[1.75]" aria-hidden="true" />
            </Button>

            <Input
              type="date"
              value={selectedDate}
              onChange={(e) => {
                if (e.target.value) {
                  updateDate(e.target.value);
                }
              }}
              aria-label={t.summary.dateInputLabel}
              className="h-10 w-auto rounded-2xl bg-card px-3 text-xs tabular-nums shadow-none"
            />

            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={() => updateDate(shiftDateString(selectedDate, 1))}
              aria-label={t.summary.nextDayLabel}
              className="size-10 rounded-full shadow-none"
            >
              <ChevronRight className="size-4 stroke-[1.75]" aria-hidden="true" />
            </Button>

            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={selectedDate === todayStr}
              onClick={() => updateDate(todayStr)}
              className="h-10 rounded-full px-4 text-xs shadow-none"
            >
              {t.summary.todayButton}
            </Button>
          </div>
        </div>
      </Card>

      {fetchError ? (
        <div className="space-y-3">
          <Alert variant="error">{fetchError}</Alert>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={triggerReload}
            className="h-10 rounded-full px-4 text-xs shadow-none"
          >
            <RotateCcw className="size-3.5 stroke-[1.75]" aria-hidden="true" />
            <span>{t.summary.retryButton}</span>
          </Button>
        </div>
      ) : isFetching && !summary ? (
        <Card className="rounded-3xl border border-border bg-card p-6 text-sm text-muted-foreground shadow-none ring-0">
          {t.common.loading}
        </Card>
      ) : (
        <>
          <section className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-3">
              <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                {t.summary.caloriesHeading}
              </p>
              <div className="flex items-baseline justify-between gap-2">
                <p className="text-xl font-semibold tabular-nums text-foreground">
                  {totals?.calories ?? 0} {t.profile.kcalUnit}
                </p>
                <span className="text-xs font-semibold tabular-nums text-[var(--brand-primary)]">
                  {progress?.calories ?? 0}%
                </span>
              </div>
              <p className="text-xs text-muted-foreground">
                {t.summary.ofTarget}{" "}
                <span className="font-medium tabular-nums text-foreground">
                  {targets?.daily_calorie_target ?? 0} {t.profile.kcalUnit}
                </span>
              </p>
              <MacroProgressBar
                label={t.summary.caloriesHeading}
                percent={progress?.calories ?? 0}
                colorVar="var(--macro-calories)"
              />
            </Card>

            <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-2.5">
              <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                {t.summary.spendingHeading}
              </p>
              <p className="text-xl font-semibold tabular-nums text-[var(--brand-accent)]">
                {formatIDR(totals?.total_spending_idr ?? 0)}
              </p>
              <p className="text-xs text-muted-foreground">
                {(totals?.meal_count ?? 0) > 0
                  ? `${totals?.meal_count ?? 0} ${t.meals.mealCountUnit}`
                  : t.summary.noMealsCount}
              </p>
            </Card>

            <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-2.5">
              <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                {t.summary.healthScoreHeading}
              </p>
              {(totals?.meal_count ?? 0) > 0 && totals?.health_label ? (
                <div className="space-y-2">
                  <p className="text-xl font-semibold tabular-nums text-foreground">
                    {totals.average_health_score}/100
                  </p>
                  <Badge
                    className={`h-6 rounded-full border-0 px-3 text-xs font-semibold ${getHealthBadgeClass(totals.health_label)}`}
                  >
                    {dailyHealthLabelText}
                  </Badge>
                </div>
              ) : (
                <p className="text-xl font-semibold tabular-nums text-muted-foreground">
                  -
                </p>
              )}
            </Card>
          </section>

          <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-4">
            <h2 className="text-base font-semibold text-foreground">
              {t.summary.macrosHeading}
            </h2>

            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {macroRows.map((macro) => (
                <div
                  key={macro.key}
                  className="rounded-2xl bg-muted p-3.5 space-y-2"
                >
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-xs font-medium text-muted-foreground">
                      {macro.label}
                    </p>
                    <span className="text-xs font-semibold tabular-nums text-foreground">
                      {macro.percent}%
                    </span>
                  </div>
                  <p className="text-sm font-semibold tabular-nums text-foreground">
                    {formatMacro(macro.actual)} / {formatMacro(macro.target)}{" "}
                    {t.profile.gramUnit}
                  </p>
                  <MacroProgressBar
                    label={macro.label}
                    percent={macro.percent}
                    colorVar={macro.colorVar}
                  />
                </div>
              ))}
            </div>
          </Card>

          <section className="space-y-3">
            <div className="flex items-center justify-between gap-2 px-1">
              <h2 className="text-base font-semibold text-foreground">
                {t.summary.savedMealsHeading}
              </h2>
              <span className="text-xs tabular-nums text-muted-foreground">
                {meals.length} {t.meals.mealCountUnit}
              </span>
            </div>

            {deleteError ? <Alert variant="error">{deleteError}</Alert> : null}

            {meals.length === 0 ? (
              <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-3">
                <p className="text-sm font-semibold text-foreground">
                  {t.summary.emptyTitle}
                </p>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  {t.summary.emptyDescription}
                </p>
                <div className="pt-1">
                  <Button
                    render={<Link href="/chat" />}
                    size="sm"
                    className="h-10 rounded-full px-5 text-xs shadow-none"
                  >
                    {t.summary.goToChatButton}
                  </Button>
                </div>
              </Card>
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
        <Card className="rounded-3xl border border-border bg-card p-6 text-sm text-muted-foreground shadow-none ring-0">
          Loading...
        </Card>
      }
    >
      <SummaryContent />
    </Suspense>
  );
}
