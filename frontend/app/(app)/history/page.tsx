"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ChevronDown, ChevronUp, RotateCcw } from "lucide-react";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Separator } from "@/components/ui/separator";
import { formatIDR, formatMacro } from "@/components/chat/format";
import { getHealthBadgeClass } from "@/components/chat/MealResultCard";
import { SavedMealItem } from "@/components/meals/SavedMealItem";
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

  if (
    isLoading ||
    !isAuthenticated ||
    !profile ||
    !profile.onboarding_completed
  ) {
    return (
      <Card className="rounded-3xl border border-border bg-card p-6 text-[15px] text-muted-foreground shadow-none ring-0">
        {t.common.loading}
      </Card>
    );
  }

  const rangeOptions: { value: HistoryRange; label: string }[] = [
    { value: 7, label: t.history.range7Days },
    { value: 14, label: t.history.range14Days },
    { value: 30, label: t.history.range30Days },
  ];

  const days = history?.days ?? [];
  const weeklyTrend = history?.weekly_trend ?? [];

  const rangeSelector = (
    <div
      role="group"
      aria-label={t.history.rangeLabel}
      className="flex flex-row items-center gap-2"
    >
      {rangeOptions.map((option) => {
        const active = daysRange === option.value;
        return (
          <Button
            key={option.value}
            type="button"
            size="sm"
            variant={active ? "secondary" : "outline"}
            aria-pressed={active}
            onClick={() => setDaysRange(option.value)}
            className={`h-11 rounded-full px-4 text-[13px] shadow-none transition-colors duration-150 ${
              active
                ? "border-transparent bg-[var(--brand-tint)] font-semibold text-[var(--brand-primary)]"
                : "font-medium"
            }`}
          >
            {option.label}
          </Button>
        );
      })}
    </div>
  );

  return (
    <div className="space-y-6">
      <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 sm:p-6">
        <div className="space-y-1">
          <h1 className="text-[22px] font-semibold leading-[1.25] tracking-[-0.015em] text-foreground sm:text-[26px]">
            {t.history.title}
          </h1>
          <p className="text-[13px] leading-[1.5] text-muted-foreground sm:text-[15px]">
            {t.history.subtitle}
          </p>
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
            className="h-11 rounded-full px-5 text-[13px] font-medium shadow-none transition-colors duration-150"
          >
            <RotateCcw className="size-3.5 stroke-[1.75]" aria-hidden="true" />
            <span>{t.history.retryButton}</span>
          </Button>
        </div>
      ) : isFetching && !history ? (
        <Card className="rounded-3xl border border-border bg-card p-6 text-[15px] text-muted-foreground shadow-none ring-0">
          {t.common.loading}
        </Card>
      ) : (
        <>
          {weeklyTrend.length > 0 ? (
            <WeeklyHealthChart points={weeklyTrend} />
          ) : null}

          <section className="space-y-3.5">
            <div className="flex flex-wrap items-center justify-between gap-3 px-1">
              <h2 className="text-[18px] font-semibold leading-[1.3] tracking-[-0.01em] text-foreground sm:text-[20px]">
                {t.history.dailyListHeading}
              </h2>
              {rangeSelector}
            </div>

            {days.length === 0 ? (
              <Card className="gap-3 rounded-3xl border border-border bg-card p-6 shadow-none ring-0 sm:p-7">
                <p className="text-base font-semibold leading-[1.35] text-foreground">
                  {t.history.emptyTitle}
                </p>
                <p className="text-[15px] leading-[1.6] text-muted-foreground">
                  {t.history.emptyDescription}
                </p>
                <div className="pt-1">
                  <Button
                    render={<Link href="/chat" />}
                    size="sm"
                    className="h-11 rounded-full px-5 text-[13px] font-semibold shadow-none transition-colors duration-150"
                  >
                    {t.history.goToChatButton}
                  </Button>
                </div>
              </Card>
            ) : (
              <div className="space-y-3.5">
                {days.map((day) => {
                  const isExpanded = Boolean(expandedDates[day.date]);
                  const dayHealthLabel = Object.hasOwn(
                    t.chat.healthLabels,
                    day.health_label
                  )
                    ? t.chat.healthLabels[day.health_label]
                    : t.chat.healthLabels.fair;
                  const hasSpending = day.total_spending_idr > 0;

                  return (
                    <Card
                      key={day.date}
                      className="gap-4 rounded-3xl border border-border bg-card p-5 shadow-none ring-0 sm:p-6"
                    >
                      <Collapsible
                        open={isExpanded}
                        onOpenChange={(open) =>
                          setExpandedDates((prev) => ({
                            ...prev,
                            [day.date]: open,
                          }))
                        }
                        className="space-y-4"
                      >
                        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                          <div className="space-y-1">
                            <h3 className="text-base font-semibold leading-[1.35] text-foreground sm:text-[18px]">
                              {formatDisplayDate(day.date, language)}
                            </h3>
                            <p className="text-[13px] leading-[1.5] tabular-nums text-muted-foreground">
                              {day.date} ·{" "}
                              <span className="font-medium text-foreground">
                                {day.meal_count} {t.meals.mealCountUnit}
                              </span>
                            </p>
                          </div>

                          <div className="flex flex-wrap items-center gap-2">
                            <CollapsibleTrigger
                              render={
                                <Button
                                  type="button"
                                  variant={isExpanded ? "secondary" : "outline"}
                                  size="sm"
                                  className={`h-11 rounded-full px-4 text-[13px] shadow-none transition-colors duration-150 ${
                                    isExpanded
                                      ? "border-transparent bg-[var(--brand-tint)] font-semibold text-[var(--brand-primary)]"
                                      : "font-medium"
                                  }`}
                                />
                              }
                            >
                              {isExpanded ? (
                                <ChevronUp
                                  className="size-4 shrink-0 stroke-[1.75]"
                                  aria-hidden="true"
                                />
                              ) : (
                                <ChevronDown
                                  className="size-4 shrink-0 stroke-[1.75]"
                                  aria-hidden="true"
                                />
                              )}
                              <span>
                                {isExpanded
                                  ? t.history.collapseMealsButton
                                  : t.history.expandMealsButton}
                              </span>
                            </CollapsibleTrigger>

                            <Button
                              render={
                                <Link
                                  href={`/summary?date=${encodeURIComponent(day.date)}`}
                                />
                              }
                              variant="outline"
                              size="sm"
                              className="h-11 rounded-full px-4 text-[13px] font-medium shadow-none transition-colors duration-150"
                            >
                              {t.history.viewSummaryLink}
                            </Button>
                          </div>
                        </div>

                        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                          <div className="flex flex-col justify-between gap-2 rounded-2xl bg-muted p-3.5">
                            <div className="space-y-1">
                              <p className="text-[11px] font-semibold uppercase leading-[1.4] tracking-[0.06em] text-muted-foreground">
                                {t.history.totalCaloriesLabel}
                              </p>
                              <p className="text-lg font-semibold leading-tight tabular-nums text-foreground">
                                {day.total_calories}{" "}
                                <span className="text-[13px] font-medium text-muted-foreground">
                                  {t.profile.kcalUnit}
                                </span>
                              </p>
                            </div>
                            <p className="text-xs tabular-nums text-muted-foreground">
                              {t.profile.protein}{" "}
                              <span className="font-medium text-foreground">
                                {formatMacro(day.total_protein_g)}
                                {t.profile.gramUnit}
                              </span>{" "}
                              · {t.profile.carbs}{" "}
                              <span className="font-medium text-foreground">
                                {formatMacro(day.total_carbs_g)}
                                {t.profile.gramUnit}
                              </span>{" "}
                              · {t.profile.fat}{" "}
                              <span className="font-medium text-foreground">
                                {formatMacro(day.total_fat_g)}
                                {t.profile.gramUnit}
                              </span>
                            </p>
                          </div>

                          <div
                            className={`flex flex-col justify-between gap-2 rounded-2xl p-3.5 ${
                              hasSpending
                                ? "bg-[var(--brand-accent-tint)]"
                                : "bg-muted"
                            }`}
                          >
                            <p className="text-[11px] font-semibold uppercase leading-[1.4] tracking-[0.06em] text-muted-foreground">
                              {t.history.totalSpendingLabel}
                            </p>
                            <p className="text-lg font-semibold leading-tight tabular-nums text-[var(--brand-accent)]">
                              {formatIDR(day.total_spending_idr)}
                            </p>
                          </div>

                          <div className="flex flex-col justify-between gap-2 rounded-2xl bg-muted p-3.5">
                            <p className="text-[11px] font-semibold uppercase leading-[1.4] tracking-[0.06em] text-muted-foreground">
                              {t.history.averageScoreLabel}
                            </p>
                            <div className="flex items-center justify-between gap-2">
                              <p className="text-lg font-semibold leading-tight tabular-nums text-foreground">
                                {day.average_health_score}
                                <span className="text-xs font-normal text-muted-foreground">
                                  /100
                                </span>
                              </p>
                              <Badge
                                className={`h-6 rounded-full border-0 px-2.5 text-xs font-semibold tabular-nums ${getHealthBadgeClass(day.health_label)}`}
                              >
                                {dayHealthLabel}
                              </Badge>
                            </div>
                          </div>
                        </div>

                        <CollapsibleContent className="space-y-4 pt-1">
                          <Separator className="bg-border" />
                          <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2">
                            {day.meals.map((meal) => (
                              <SavedMealItem key={meal.id} meal={meal} />
                            ))}
                          </div>
                        </CollapsibleContent>
                      </Collapsible>
                    </Card>
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
