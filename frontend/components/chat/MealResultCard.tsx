"use client";

import { useState } from "react";
import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import {
  Beef,
  Bookmark,
  Camera,
  Check,
  ChevronRight,
  Droplet,
  Flame,
  Leaf,
  Loader2,
  ShoppingBasket,
  Utensils,
  Wallet,
  Wheat,
  X,
} from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { formatIDR, formatMacro } from "@/components/chat/format";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DatePicker } from "@/components/ui/DatePicker";
import {
  Dialog,
  DialogClose,
  DialogDescription,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { getLocalDateString, shiftDateString } from "@/lib/date";
import type {
  HealthLabel,
  InputSource,
  MealOption,
  MealTime,
} from "@/lib/types";

interface MealResultCardProps {
  meal: MealOption;
  onSave?: (
    meal: MealOption,
    mealTime?: MealTime,
    mealDate?: string
  ) => void | Promise<void>;
  saved?: boolean;
  isSaving?: boolean;
  saveError?: string | null;
}

function SourceIcon({ source }: { source: InputSource }) {
  const iconClass = "size-4 shrink-0 stroke-[1.75] text-muted-foreground";
  if (source === "budget") {
    return <ShoppingBasket className={iconClass} aria-hidden="true" />;
  }
  if (source === "photo") {
    return <Camera className={iconClass} aria-hidden="true" />;
  }
  return <Utensils className={iconClass} aria-hidden="true" />;
}

export function getHealthBadgeClass(label: HealthLabel): string {
  if (label === "good") {
    return "bg-[var(--score-good-bg)] text-[var(--score-good-text)]";
  }
  if (label === "fair") {
    return "bg-[var(--score-fair-bg)] text-[var(--score-fair-text)]";
  }
  return "bg-[var(--score-low-bg)] text-[var(--score-low-text)]";
}

export function MealResultCard({
  meal,
  onSave,
  saved = false,
  isSaving = false,
  saveError = null,
}: MealResultCardProps) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [selectedMealTime, setSelectedMealTime] = useState<"auto" | MealTime>(
    "auto"
  );
  const todayStr = getLocalDateString();
  const maxDateStr = shiftDateString(todayStr, 30);
  const [selectedMealDate, setSelectedMealDate] = useState<string>(todayStr);

  const hasMissing =
    Array.isArray(meal.missing_ingredients) &&
    meal.missing_ingredients.length > 0;
  const hasSteps =
    Array.isArray(meal.cooking_steps) && meal.cooking_steps.length > 0;
  const hasDetails = hasMissing || hasSteps;

  const healthLabelText = Object.hasOwn(t.chat.healthLabels, meal.health_label)
    ? t.chat.healthLabels[meal.health_label]
    : t.chat.healthLabels.fair;

  const mealTimeLabels: Record<"auto" | MealTime, string> = {
    auto: t.meals.mealTimeAuto,
    breakfast: t.meals.mealTimes.breakfast,
    lunch: t.meals.mealTimes.lunch,
    dinner: t.meals.mealTimes.dinner,
    snack: t.meals.mealTimes.snack,
  };

  const macros = [
    {
      label: t.chat.caloriesLabel,
      value: `${Math.round(meal.calories)} ${t.profile.kcalUnit}`,
      icon: Flame,
      colorVar: "var(--macro-calories)",
    },
    {
      label: t.chat.proteinLabel,
      value: `${formatMacro(meal.protein_g)} ${t.profile.gramUnit}`,
      icon: Beef,
      colorVar: "var(--macro-protein)",
    },
    {
      label: t.chat.carbsLabel,
      value: `${formatMacro(meal.carbs_g)} ${t.profile.gramUnit}`,
      icon: Wheat,
      colorVar: "var(--macro-carbs)",
    },
    {
      label: t.chat.fatLabel,
      value: `${formatMacro(meal.fat_g)} ${t.profile.gramUnit}`,
      icon: Droplet,
      colorVar: "var(--macro-fat)",
    },
    {
      label: t.chat.fiberLabel,
      value: `${formatMacro(meal.fiber_g)} ${t.profile.gramUnit}`,
      icon: Leaf,
      colorVar: "var(--macro-fiber)",
    },
  ];

  const compactSummaryText = meal.note || meal.portion || "";

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          <button
            type="button"
            className="group w-full cursor-pointer rounded-2xl border border-border bg-card px-4 py-3.5 text-left shadow-none transition-colors duration-150 hover:border-[var(--border-strong)] hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)]"
          />
        }
      >
        <div className="space-y-1.5">
          {/* Row 1: Title + Health Score Badge */}
          <div className="flex items-center justify-between gap-2.5">
            <div className="flex min-w-0 items-center gap-2">
              <SourceIcon source={meal.input_source} />
              <h3 className="truncate text-sm font-semibold leading-snug text-foreground sm:text-[15px]">
                {meal.name}
              </h3>
            </div>

            <Badge
              className={`h-6 shrink-0 rounded-full border-0 px-2.5 text-[11px] font-semibold tabular-nums ${getHealthBadgeClass(meal.health_label)}`}
            >
              {meal.health_score} · {healthLabelText}
            </Badge>
          </div>

          {/* Row 2: Key metrics (Calories, Protein, Price) */}
          <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs tabular-nums text-muted-foreground">
            <span className="font-medium text-foreground">
              {Math.round(meal.calories)} {t.profile.kcalUnit}
            </span>
            <span aria-hidden="true">·</span>
            <span>
              {t.chat.proteinLabel} {formatMacro(meal.protein_g)}{" "}
              {t.profile.gramUnit}
            </span>
            {meal.estimated_price_idr > 0 ? (
              <>
                <span aria-hidden="true">·</span>
                <span className="font-semibold text-[var(--brand-accent)]">
                  {formatIDR(meal.estimated_price_idr)}
                </span>
              </>
            ) : null}
          </div>

          {/* Row 3: Note/Portion preview or Saved indicator + Details cue */}
          <div className="flex items-center justify-between gap-3 pt-0.5">
            {saved ? (
              <span className="inline-flex items-center gap-1 text-xs font-medium text-[var(--brand-primary)]">
                <Check className="size-3.5 stroke-[2]" aria-hidden="true" />
                <span>{t.chat.savedMealButton}</span>
              </span>
            ) : (
              <p className="min-w-0 truncate text-xs text-muted-foreground">
                {compactSummaryText}
              </p>
            )}

            <span className="inline-flex shrink-0 items-center gap-0.5 text-xs font-medium text-[var(--brand-primary)] transition-transform duration-150 group-hover:translate-x-0.5">
              <span>{t.chat.openMealDetailLabel}</span>
              <ChevronRight
                className="size-3.5 stroke-[2]"
                aria-hidden="true"
              />
            </span>
          </div>
        </div>
      </DialogTrigger>

      <DialogPortal>
        <DialogOverlay className="overflow-hidden overscroll-contain bg-black/40 supports-backdrop-filter:backdrop-blur-md" />
        <DialogPrimitive.Popup
          data-slot="dialog-content"
          className="fixed top-1/2 left-1/2 z-50 grid max-h-[88dvh] w-full max-w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 gap-4 overflow-y-auto overscroll-contain rounded-3xl border border-border bg-card p-5 text-sm text-card-foreground shadow-none ring-0 outline-none duration-150 sm:max-w-lg sm:p-6 data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95"
        >
          <DialogHeader className="gap-2 pr-9">
            <div className="flex flex-wrap items-start justify-between gap-2.5">
              <div className="min-w-0 space-y-1">
                <div className="flex items-center gap-2">
                  <SourceIcon source={meal.input_source} />
                  <DialogTitle className="text-base font-semibold leading-snug text-foreground sm:text-lg">
                    {meal.name}
                  </DialogTitle>
                </div>
                {meal.portion ? (
                  <DialogDescription className="text-xs text-muted-foreground">
                    {meal.portion}
                  </DialogDescription>
                ) : (
                  <DialogDescription className="sr-only">
                    {meal.name}
                  </DialogDescription>
                )}
              </div>

              <Badge
                className={`h-7 shrink-0 rounded-full border-0 px-3 text-xs font-semibold tabular-nums ${getHealthBadgeClass(meal.health_label)}`}
              >
                {meal.health_score} · {healthLabelText}
              </Badge>
            </div>
          </DialogHeader>

          <DialogClose
            render={
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t.chat.closeMealDetailLabel}
                className="absolute top-4 right-4 size-8 rounded-full text-muted-foreground shadow-none hover:bg-accent hover:text-foreground"
              />
            }
          >
            <X className="size-4 stroke-[1.75]" aria-hidden="true" />
          </DialogClose>

          <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
            {meal.estimated_price_idr > 0 ? (
              <div className="flex items-center gap-1.5">
                <Wallet
                  className="size-3.5 shrink-0 stroke-[1.75] text-[var(--brand-accent)]"
                  aria-hidden="true"
                />
                <span className="text-xs text-muted-foreground">
                  {t.chat.priceLabel}:
                </span>
                <span className="font-semibold tabular-nums text-[var(--brand-accent)]">
                  {formatIDR(meal.estimated_price_idr)}
                </span>
              </div>
            ) : (
              <span />
            )}

            <span className="text-xs text-[var(--text-muted)]">
              {t.chat.estimatedTag}
            </span>
          </div>

          <div className="grid grid-cols-2 gap-2 rounded-2xl bg-muted p-3 sm:grid-cols-5">
            {macros.map((macro) => {
              const Icon = macro.icon;
              return (
                <div key={macro.label} className="space-y-0.5 px-1">
                  <p className="flex items-center gap-1 text-[11px] text-muted-foreground">
                    <Icon
                      className="size-3.5 shrink-0 stroke-[1.75]"
                      style={{ color: macro.colorVar }}
                      aria-hidden="true"
                    />
                    <span>{macro.label}</span>
                  </p>
                  <p className="whitespace-nowrap text-sm font-semibold tabular-nums text-foreground">
                    {macro.value}
                  </p>
                </div>
              );
            })}
          </div>

          {meal.note ? (
            <p className="text-sm leading-relaxed text-muted-foreground">
              {meal.note}
            </p>
          ) : null}

          {hasDetails ? (
            <div className="space-y-3 rounded-2xl bg-muted p-4 text-sm">
              {hasMissing ? (
                <div className="space-y-1.5">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                    {t.chat.missingIngredientsHeading}
                  </p>
                  <ul className="list-disc space-y-1 pl-5 text-foreground">
                    {meal.missing_ingredients.map((item, idx) => (
                      <li key={`${item}-${idx}`}>{item}</li>
                    ))}
                  </ul>
                </div>
              ) : null}

              {hasSteps ? (
                <div className="space-y-1.5">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                    {t.chat.cookingStepsHeading}
                  </p>
                  <ol className="list-decimal space-y-1 pl-5 text-foreground">
                    {meal.cooking_steps.map((step, idx) => (
                      <li key={`${step}-${idx}`}>{step}</li>
                    ))}
                  </ol>
                </div>
              ) : null}
            </div>
          ) : null}

          <Separator className="bg-border" />

          <div className="space-y-2.5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex flex-wrap items-center gap-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground">
                    {t.chat.mealDateSelectLabel}
                  </span>
                  <DatePicker
                    value={selectedMealDate}
                    minDate={todayStr}
                    maxDate={maxDateStr}
                    disabled={saved || isSaving}
                    onChange={setSelectedMealDate}
                    ariaLabel={t.chat.mealDateSelectLabel}
                    size="sm"
                  />
                </div>

                <div className="flex items-center gap-2">
                  <span className="text-xs text-muted-foreground">
                    {t.chat.mealTimeSelectLabel}
                  </span>
                  <Select
                    value={selectedMealTime}
                    disabled={saved || isSaving}
                    onValueChange={(val) =>
                      setSelectedMealTime((val ?? "auto") as "auto" | MealTime)
                    }
                  >
                    <SelectTrigger
                      aria-label={t.chat.mealTimeSelectLabel}
                      className="h-9 min-w-28 rounded-full border-border bg-card px-3 text-xs shadow-none data-[size=default]:h-9"
                    >
                      <SelectValue>
                        {mealTimeLabels[selectedMealTime]}
                      </SelectValue>
                    </SelectTrigger>
                    <SelectContent
                      alignItemWithTrigger={false}
                      sideOffset={6}
                      className="rounded-[12px] border border-border bg-card p-1 shadow-none ring-0"
                    >
                      <SelectItem
                        value="auto"
                        className="rounded-[8px] py-1.5 pl-2.5 pr-8 text-xs"
                      >
                        {t.meals.mealTimeAuto}
                      </SelectItem>
                      <SelectItem
                        value="breakfast"
                        className="rounded-[8px] py-1.5 pl-2.5 pr-8 text-xs"
                      >
                        {t.meals.mealTimes.breakfast}
                      </SelectItem>
                      <SelectItem
                        value="lunch"
                        className="rounded-[8px] py-1.5 pl-2.5 pr-8 text-xs"
                      >
                        {t.meals.mealTimes.lunch}
                      </SelectItem>
                      <SelectItem
                        value="dinner"
                        className="rounded-[8px] py-1.5 pl-2.5 pr-8 text-xs"
                      >
                        {t.meals.mealTimes.dinner}
                      </SelectItem>
                      <SelectItem
                        value="snack"
                        className="rounded-[8px] py-1.5 pl-2.5 pr-8 text-xs"
                      >
                        {t.meals.mealTimes.snack}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <Button
                type="button"
                variant={saved ? "secondary" : "default"}
                disabled={saved || isSaving}
                onClick={() =>
                  void onSave?.(
                    meal,
                    selectedMealTime === "auto" ? undefined : selectedMealTime,
                    selectedMealDate
                  )
                }
                aria-pressed={saved}
                className={`h-9 rounded-full px-4 text-xs font-medium shadow-none ${
                  saved
                    ? "bg-[var(--brand-tint)] text-[var(--brand-primary)] opacity-100"
                    : ""
                }`}
              >
                {isSaving ? (
                  <Loader2
                    className="size-3.5 shrink-0 animate-spin stroke-[1.75]"
                    aria-hidden="true"
                  />
                ) : saved ? (
                  <Check
                    className="size-3.5 shrink-0 stroke-[1.75]"
                    aria-hidden="true"
                  />
                ) : (
                  <Bookmark
                    className="size-3.5 shrink-0 stroke-[1.75]"
                    aria-hidden="true"
                  />
                )}
                <span>
                  {isSaving
                    ? t.chat.savingMealButton
                    : saved
                      ? t.chat.savedMealButton
                      : t.chat.saveMealButton}
                </span>
              </Button>
            </div>

            {saveError ? (
              <p role="alert" className="text-xs text-[var(--error-text)]">
                {saveError}
              </p>
            ) : null}
          </div>
        </DialogPrimitive.Popup>
      </DialogPortal>
    </Dialog>
  );
}
