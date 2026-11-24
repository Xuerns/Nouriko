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
import { formatIDR } from "@/components/chat/format";
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
      <Card className="rounded-3xl border border-border bg-card p-6 text-sm text-muted-foreground shadow-none ring-0">
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

  return (
    <div className="space-y-5">
      <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="space-y-1">
            <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground sm:text-[26px]">
              {t.history.title}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t.history.subtitle}
            </p>
          </div>

          <div
            role="group"
            aria-label={t.history.rangeLabel}
            className="flex flex-wrap items-center gap-1.5"
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
                  className={`h-9 rounded-full px-4 text-xs shadow-none ${
                    active
                      ? "bg-[var(--brand-tint)] font-semibold text-[var(--brand-primary)]"
                      : ""
                  }`}
                >
                  {option.label}
                </Button>
              );
            })}
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
            <span>{t.history.retryButton}</span>
          </Button>
        </div>
      ) : isFetching && !history ? (
        <Card className="rounded-3xl border border-border bg-card p-6 text-sm text-muted-foreground shadow-none ring-0">
          {t.common.loading}
        </Card>
      ) : (
        <>
          {weeklyTrend.length > 0 ? (
            <WeeklyHealthChart points={weeklyTrend} />
          ) : null}

          <section className="space-y-3">
            <h2 className="px-1 text-base font-semibold text-foreground">
              {t.history.dailyListHeading}
            </h2>

            {days.length === 0 ? (
              <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-3">
                <p className="text-sm font-semibold text-foreground">
                  {t.history.emptyTitle}
                </p>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  {t.history.emptyDescription}
                </p>
                <div className="pt-1">
                  <Button
                    render={<Link href="/chat" />}
                    size="sm"
                    className="h-10 rounded-full px-5 text-xs shadow-none"
                  >
                    {t.history.goToChatButton}
                  </Button>
                </div>
              </Card>
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
                    <Card
                      key={day.date}
                      className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-4"
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
                        <div className="flex flex-wrap items-start justify-between gap-3">
                          <div className="space-y-0.5">
                            <h3 className="text-base font-semibold text-foreground">
                              {formatDisplayDate(day.date, language)}
                            </h3>
                            <p className="text-xs tabular-nums text-muted-foreground">
                              {day.date} · {day.meal_count}{" "}
                              {t.meals.mealCountUnit}
                            </p>
                          </div>

                          <div className="flex flex-wrap items-center gap-2">
                            <CollapsibleTrigger
                              render={
                                <Button
                                  type="button"
                                  variant="outline"
                                  size="sm"
                                  className="h-9 rounded-full px-3.5 text-xs shadow-none"
                                />
                              }
                            >
                              {isExpanded ? (
                                <ChevronUp
                                  className="size-3.5 shrink-0 stroke-[1.75]"
                                  aria-hidden="true"
                                />
                              ) : (
                                <ChevronDown
                                  className="size-3.5 shrink-0 stroke-[1.75]"
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
                              className="h-9 rounded-full px-3.5 text-xs shadow-none"
                            >
                              {t.history.viewSummaryLink}
                            </Button>
                          </div>
                        </div>

                        <div className="grid grid-cols-1 gap-2 rounded-2xl bg-muted p-3 sm:grid-cols-3">
                          <div className="px-1">
                            <p className="text-xs text-muted-foreground">
                              {t.history.totalCaloriesLabel}
                            </p>
                            <p className="mt-0.5 text-sm font-semibold tabular-nums text-foreground">
                              {day.total_calories} {t.profile.kcalUnit}
                            </p>
                          </div>

                          <div className="px-1">
                            <p className="text-xs text-muted-foreground">
                              {t.history.totalSpendingLabel}
                            </p>
                            <p className="mt-0.5 text-sm font-semibold tabular-nums text-[var(--brand-accent)]">
                              {formatIDR(day.total_spending_idr)}
                            </p>
                          </div>

                          <div className="px-1">
                            <p className="text-xs text-muted-foreground">
                              {t.history.averageScoreLabel}
                            </p>
                            <div className="mt-0.5 flex items-center gap-2">
                              <Badge
                                className={`h-6 rounded-full border-0 px-2.5 text-xs font-semibold tabular-nums ${getHealthBadgeClass(day.health_label)}`}
                              >
                                {day.average_health_score} · {dayHealthLabel}
                              </Badge>
                            </div>
                          </div>
                        </div>

                        <CollapsibleContent className="space-y-3">
                          <Separator className="bg-border" />
                          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
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
