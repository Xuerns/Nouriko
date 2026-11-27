"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import {
  Beef,
  Check,
  ChevronLeft,
  ChevronRight,
  Cookie,
  Droplet,
  Flame,
  HeartPulse,
  Leaf,
  Moon,
  RotateCcw,
  Sun,
  Sunrise,
  Wallet,
  Wheat,
} from "lucide-react";
import { NourikoMark } from "@/components/layout/NourikoMark";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { DatePicker } from "@/components/ui/DatePicker";
import { SummaryPageSkeleton } from "@/components/ui/PageSkeletons";
import { Separator } from "@/components/ui/separator";
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
import type { DailySummaryResponse, MealTime } from "@/lib/types";

const MEAL_TIME_ORDER: MealTime[] = ["breakfast", "lunch", "dinner", "snack"];

const MEAL_TIME_ICONS = {
  breakfast: Sunrise,
  lunch: Sun,
  dinner: Moon,
  snack: Cookie,
} as const;

function MacroProgressBar({
  label,
  percent,
  colorVar,
  heightClass = "h-2.5",
}: {
  label: string;
  percent: number;
  colorVar: string;
  heightClass?: string;
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
      className={`${heightClass} w-full overflow-hidden rounded-full bg-muted`}
    >
      <div
        className="h-full rounded-full transition-all duration-200"
        style={{ width: `${fillWidth}%`, backgroundColor: colorVar }}
      />
    </div>
  );
}

function SummaryLoadingState({ label }: { label: string }) {
  return <SummaryPageSkeleton label={label} />;
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
  const [selectedSlot, setSelectedSlot] = useState<"all" | MealTime>("all");

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
    setSelectedSlot("all");
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
    return <SummaryLoadingState label={t.common.loading} />;
  }

  const totals = summary?.totals;
  const targets = summary?.targets;
  const progress = summary?.progress_percent;
  const meals = summary?.meals ?? [];

  const targetCalories = targets?.daily_calorie_target ?? 0;
  const actualCalories = totals?.calories ?? 0;
  const remainingCalories = targetCalories - actualCalories;
  const remainingProtein = Math.max(
    0,
    Math.round((targets?.target_protein_g ?? 0) - (totals?.protein_g ?? 0))
  );
  const remainingFiber = Math.max(
    0,
    Math.round((targets?.target_fiber_g ?? 0) - (totals?.fiber_g ?? 0))
  );

  const missingMacroParts: string[] = [];
  if (remainingProtein > 0) {
    missingMacroParts.push(
      `${remainingProtein} ${t.profile.gramUnit} ${t.profile.protein.toLowerCase()}`
    );
  }
  if (remainingFiber > 0) {
    missingMacroParts.push(
      `${remainingFiber} ${t.profile.gramUnit} ${t.profile.fiber.toLowerCase()}`
    );
  }

  const remainingPrompt =
    remainingCalories > 0
      ? language === "en"
        ? `Recommend healthy meals for my remaining daily target: around ${remainingCalories} kcal${
            missingMacroParts.length > 0
              ? `, ${missingMacroParts.join(" and ")}`
              : ""
          }.`
        : `Rekomendasikan menu bergizi untuk sisa target hari ini: sekitar ${remainingCalories} kkal${
            missingMacroParts.length > 0
              ? `, ${missingMacroParts.join(" dan ")}`
              : ""
          }.`
      : language === "en"
        ? "Suggest light, low-calorie healthy snack or meal options."
        : "Rekomendasikan pilihan menu atau camilan sehat yang ringan dan rendah kalori.";

  const remainingChatHref = `/chat?prompt=${encodeURIComponent(remainingPrompt)}`;

  const slotSummaries = MEAL_TIME_ORDER.map((slot) => {
    const slotMeals = meals.filter((m) => m.meal_time === slot);
    const slotCalories = slotMeals.reduce(
      (sum, m) => sum + Math.round(m.calories),
      0
    );
    return {
      slot,
      label: t.meals.mealTimes[slot],
      count: slotMeals.length,
      calories: slotCalories,
    };
  });

  const filteredMeals =
    selectedSlot === "all"
      ? meals
      : meals.filter((m) => m.meal_time === selectedSlot);

  const activeSlotLabel =
    selectedSlot === "all" ? "" : t.meals.mealTimes[selectedSlot];

  const activeSlotChatHref =
    selectedSlot === "all"
      ? "/chat"
      : `/chat?prompt=${encodeURIComponent(
          language === "en"
            ? `Recommend healthy ${activeSlotLabel.toLowerCase()} meal ideas for today.`
            : `Rekomendasikan ide menu ${activeSlotLabel.toLowerCase()} yang bergizi untuk hari ini.`
        )}`;

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
      icon: Beef,
    },
    {
      key: "carbs",
      label: t.profile.carbs,
      actual: totals?.carbs_g ?? 0,
      target: targets?.target_carbs_g ?? 0,
      percent: progress?.carbs_g ?? 0,
      colorVar: "var(--macro-carbs)",
      icon: Wheat,
    },
    {
      key: "fat",
      label: t.profile.fat,
      actual: totals?.fat_g ?? 0,
      target: targets?.target_fat_g ?? 0,
      percent: progress?.fat_g ?? 0,
      colorVar: "var(--macro-fat)",
      icon: Droplet,
    },
    {
      key: "fiber",
      label: t.profile.fiber,
      actual: totals?.fiber_g ?? 0,
      target: targets?.target_fiber_g ?? 0,
      percent: progress?.fiber_g ?? 0,
      colorVar: "var(--macro-fiber)",
      icon: Leaf,
    },
  ];

  return (
    <div className="space-y-8">
      <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1">
          <h1 className="text-[24px] font-semibold tracking-[-0.015em] text-foreground sm:text-[28px]">
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
            className="size-11 rounded-full bg-card shadow-none"
          >
            <ChevronLeft className="size-4 stroke-[1.75]" aria-hidden="true" />
          </Button>

          <DatePicker
            value={selectedDate}
            onChange={updateDate}
            ariaLabel={t.summary.dateInputLabel}
          />

          <Button
            type="button"
            variant="outline"
            size="icon"
            onClick={() => updateDate(shiftDateString(selectedDate, 1))}
            aria-label={t.summary.nextDayLabel}
            className="size-11 rounded-full bg-card shadow-none"
          >
            <ChevronRight className="size-4 stroke-[1.75]" aria-hidden="true" />
          </Button>

          <Button
            type="button"
            variant="outline"
            disabled={selectedDate === todayStr}
            onClick={() => updateDate(todayStr)}
            className="h-11 rounded-full bg-card px-4 text-xs font-medium shadow-none"
          >
            {t.summary.todayButton}
          </Button>
        </div>
      </header>

      {fetchError ? (
        <div className="space-y-3">
          <Alert variant="error">{fetchError}</Alert>
          <Button
            type="button"
            variant="outline"
            onClick={triggerReload}
            className="h-11 rounded-full px-5 text-xs font-medium shadow-none"
          >
            <RotateCcw className="size-4 stroke-[1.75]" aria-hidden="true" />
            <span>{t.summary.retryButton}</span>
          </Button>
        </div>
      ) : isFetching && !summary ? (
        <SummaryLoadingState label={t.common.loading} />
      ) : (
        <div className="grid grid-cols-1 gap-8 lg:grid-cols-12 lg:items-start">
          <div className="space-y-5 lg:sticky lg:top-24 lg:col-span-5">
            <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5">
              <div className="space-y-3">
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-1.5">
                    <Flame
                      className="size-4 shrink-0 stroke-[1.75] text-[var(--macro-calories)]"
                      aria-hidden="true"
                    />
                    <h2 className="text-sm font-medium text-muted-foreground">
                      {t.summary.caloriesHeading}
                    </h2>
                  </div>
                  <span className="rounded-full bg-[var(--brand-tint)] px-3 py-1 text-xs font-semibold tabular-nums text-[var(--brand-primary)]">
                    {progress?.calories ?? 0}%
                  </span>
                </div>

                <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                  <p className="text-3xl font-semibold tracking-[-0.02em] tabular-nums text-foreground">
                    {actualCalories}
                  </p>
                  <span className="text-sm text-muted-foreground">
                    {t.profile.kcalUnit} {t.summary.ofTarget}{" "}
                    <span className="font-medium tabular-nums text-foreground">
                      {targetCalories} {t.profile.kcalUnit}
                    </span>
                  </span>
                </div>

                <MacroProgressBar
                  label={t.summary.caloriesHeading}
                  percent={progress?.calories ?? 0}
                  colorVar="var(--macro-calories)"
                  heightClass="h-3"
                />

                {targetCalories > 0 ? (
                  <div className="space-y-3 pt-1">
                    {remainingCalories > 0 ? (
                      <p className="text-xs leading-relaxed text-muted-foreground">
                        {t.summary.remainingLabel}{" "}
                        <span className="font-semibold tabular-nums text-foreground">
                          {remainingCalories} {t.profile.kcalUnit}
                        </span>
                        {missingMacroParts.length > 0 ? (
                          <>
                            {" · "}
                            {t.summary.shortOnLabel}{" "}
                            <span className="font-medium tabular-nums text-foreground">
                              {missingMacroParts.join(" & ")}
                            </span>
                          </>
                        ) : null}
                      </p>
                    ) : remainingCalories < 0 ? (
                      <p className="text-xs font-medium tabular-nums text-[var(--score-fair-text)]">
                        {t.summary.overTargetLabel} +
                        {Math.abs(remainingCalories)} {t.profile.kcalUnit}
                      </p>
                    ) : (
                      <p className="text-xs font-medium text-[var(--score-good-text)]">
                        {t.summary.targetMetLabel}
                      </p>
                    )}

                    <Button
                      render={<Link href={remainingChatHref} />}
                      variant="secondary"
                      className="h-11 w-full gap-2 rounded-full bg-[var(--brand-tint)] px-4 text-xs font-semibold text-[var(--brand-primary)] hover:bg-[var(--brand-tint)]/80 shadow-none"
                    >
                      <NourikoMark
                        className="size-6 shrink-0"
                        aria-hidden="true"
                      />
                      <span>
                        {remainingCalories > 0
                          ? t.summary.findMealsForRemainingButton
                          : t.summary.askLightMealButton}
                      </span>
                    </Button>
                  </div>
                ) : null}
              </div>

              <Separator className="bg-border" />

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <p className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                    <Wallet
                      className="size-3.5 shrink-0 stroke-[1.75] text-[var(--brand-accent)]"
                      aria-hidden="true"
                    />
                    <span>{t.summary.spendingHeading}</span>
                  </p>
                  <p className="text-lg font-semibold tabular-nums text-[var(--brand-accent)]">
                    {formatIDR(totals?.total_spending_idr ?? 0)}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {(totals?.meal_count ?? 0) > 0
                      ? `${totals?.meal_count ?? 0} ${t.meals.mealCountUnit}`
                      : t.summary.noMealsCount}
                  </p>
                </div>

                <div className="space-y-1.5">
                  <p className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                    <HeartPulse
                      className="size-3.5 shrink-0 stroke-[1.75] text-[var(--brand-primary)]"
                      aria-hidden="true"
                    />
                    <span>{t.summary.healthScoreHeading}</span>
                  </p>
                  {(totals?.meal_count ?? 0) > 0 && totals?.health_label ? (
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="text-lg font-semibold tabular-nums text-foreground">
                        {totals.average_health_score}/100
                      </p>
                      <Badge
                        className={`h-6 rounded-full border-0 px-2.5 text-xs font-semibold ${getHealthBadgeClass(totals.health_label)}`}
                      >
                        {dailyHealthLabelText}
                      </Badge>
                    </div>
                  ) : (
                    <p className="text-lg font-semibold tabular-nums text-muted-foreground">
                      -
                    </p>
                  )}
                </div>
              </div>
            </Card>

            <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-4">
              <h2 className="text-base font-semibold text-foreground">
                {t.summary.macrosHeading}
              </h2>

              <div className="space-y-4">
                {macroRows.map((macro) => {
                  const Icon = macro.icon;
                  return (
                    <div key={macro.key} className="space-y-1.5">
                      <div className="flex items-center justify-between gap-3 text-sm">
                        <span className="flex items-center gap-1.5 font-medium text-foreground">
                          <Icon
                            className="size-4 shrink-0 stroke-[1.75]"
                            style={{ color: macro.colorVar }}
                            aria-hidden="true"
                          />
                          <span>{macro.label}</span>
                        </span>
                        <div className="flex items-baseline gap-2 tabular-nums">
                          <span className="text-sm font-semibold text-foreground">
                            {formatMacro(macro.actual)}{" "}
                            <span className="font-normal text-muted-foreground">
                              / {formatMacro(macro.target)} {t.profile.gramUnit}
                            </span>
                          </span>
                          <span className="min-w-9 text-right text-xs text-muted-foreground">
                            {macro.percent}%
                          </span>
                        </div>
                      </div>
                      <MacroProgressBar
                        label={macro.label}
                        percent={macro.percent}
                        colorVar={macro.colorVar}
                      />
                    </div>
                  );
                })}
              </div>
            </Card>
          </div>

          <section className="lg:col-span-7">
            <Card className="flex flex-col gap-5 rounded-3xl border border-border bg-card p-6 shadow-none ring-0">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex items-baseline gap-2.5">
                  <h2 className="text-base font-semibold text-foreground">
                    {t.summary.savedMealsHeading}
                  </h2>
                  <span className="text-xs tabular-nums text-muted-foreground">
                    {filteredMeals.length} {t.meals.mealCountUnit}
                  </span>
                </div>

                {selectedSlot !== "all" ? (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setSelectedSlot("all")}
                    className="h-9 rounded-full px-3.5 text-xs shadow-none"
                  >
                    {t.summary.allMealTimesFilter}
                  </Button>
                ) : null}
              </div>

              <div
                role="group"
                aria-label={t.chat.mealTimeSelectLabel}
                className="grid grid-cols-2 gap-2.5 sm:grid-cols-4"
              >
                {slotSummaries.map((item) => {
                  const isActive = selectedSlot === item.slot;
                  const isFilled = item.count > 0;
                  const SlotIcon = MEAL_TIME_ICONS[item.slot];
                  return (
                    <button
                      key={item.slot}
                      type="button"
                      aria-pressed={isActive}
                      onClick={() =>
                        setSelectedSlot((prev) =>
                          prev === item.slot ? "all" : item.slot
                        )
                      }
                      className={`flex min-h-16 flex-col justify-between gap-1 rounded-2xl border p-3 text-left transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)] focus-visible:ring-offset-2 ${
                        isActive
                          ? "border-[var(--brand-primary)] bg-[var(--brand-tint)]"
                          : isFilled
                            ? "border-border bg-background hover:bg-accent"
                            : "border-dashed border-border bg-background/50 hover:bg-accent"
                      }`}
                    >
                      <div className="flex items-center justify-between gap-1">
                        <span
                          className={`flex min-w-0 items-center gap-1.5 text-xs font-medium ${
                            isActive
                              ? "text-[var(--brand-primary)]"
                              : "text-foreground"
                          }`}
                        >
                          <SlotIcon
                            className={`size-3.5 shrink-0 stroke-[1.75] ${
                              isActive
                                ? "text-[var(--brand-primary)]"
                                : "text-muted-foreground"
                            }`}
                            aria-hidden="true"
                          />
                          <span className="truncate">{item.label}</span>
                        </span>
                        {isFilled ? (
                          <Check
                            className="size-3.5 shrink-0 stroke-[2] text-[var(--brand-primary)]"
                            aria-hidden="true"
                          />
                        ) : null}
                      </div>

                      {isFilled ? (
                        <p className="text-xs font-semibold tabular-nums text-foreground">
                          {item.calories}{" "}
                          <span className="font-normal text-muted-foreground">
                            {t.profile.kcalUnit}
                          </span>
                        </p>
                      ) : (
                        <p className="text-xs text-muted-foreground">
                          {t.summary.slotEmptyLabel}
                        </p>
                      )}
                    </button>
                  );
                })}
              </div>

              <Separator className="bg-border" />

              {deleteError ? <Alert variant="error">{deleteError}</Alert> : null}

              {meals.length === 0 ? (
                <div className="space-y-3 py-2">
                  <p className="text-base font-semibold text-foreground">
                    {t.summary.emptyTitle}
                  </p>
                  <p className="max-w-md text-sm leading-relaxed text-muted-foreground">
                    {t.summary.emptyDescription}
                  </p>
                  <div className="pt-2">
                    <Button
                      render={<Link href="/chat" />}
                      className="h-11 rounded-full px-6 text-sm font-medium shadow-none"
                    >
                      {t.summary.goToChatButton}
                    </Button>
                  </div>
                </div>
              ) : filteredMeals.length === 0 ? (
                <div className="space-y-3 py-2">
                  <p className="text-sm font-medium text-foreground">
                    {t.summary.slotEmptyListMessage.replace(
                      "{slot}",
                      activeSlotLabel
                    )}
                  </p>
                  <div className="flex flex-wrap items-center gap-2 pt-1">
                    <Button
                      render={<Link href={activeSlotChatHref} />}
                      size="sm"
                      className="h-11 rounded-full px-5 text-xs font-medium shadow-none"
                    >
                      {t.summary.slotFindMealButton.replace(
                        "{slot}",
                        activeSlotLabel
                      )}
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => setSelectedSlot("all")}
                      className="h-11 rounded-full px-5 text-xs font-medium shadow-none"
                    >
                      {t.summary.allMealTimesFilter}
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col gap-3.5">
                  {filteredMeals.map((meal) => (
                    <SavedMealItem
                      key={meal.id}
                      meal={meal}
                      surface="canvas"
                      onDelete={handleDeleteMeal}
                      isDeleting={Boolean(deletingIds[meal.id])}
                    />
                  ))}
                </div>
              )}
            </Card>
          </section>
        </div>
      )}
    </div>
  );
}

export default function SummaryPage() {
  return (
    <Suspense fallback={<SummaryLoadingState label="Loading..." />}>
      <SummaryContent />
    </Suspense>
  );
}
