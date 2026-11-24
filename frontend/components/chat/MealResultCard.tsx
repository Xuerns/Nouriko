"use client";

import { useState } from "react";
import {
  Bookmark,
  Camera,
  Check,
  ChevronDown,
  ChevronUp,
  Loader2,
  ShoppingBasket,
  Utensils,
} from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { formatIDR, formatMacro } from "@/components/chat/format";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Input } from "@/components/ui/input";
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
  const [detailsOpen, setDetailsOpen] = useState(false);
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
    },
    {
      label: t.chat.proteinLabel,
      value: `${formatMacro(meal.protein_g)} ${t.profile.gramUnit}`,
    },
    {
      label: t.chat.carbsLabel,
      value: `${formatMacro(meal.carbs_g)} ${t.profile.gramUnit}`,
    },
    {
      label: t.chat.fatLabel,
      value: `${formatMacro(meal.fat_g)} ${t.profile.gramUnit}`,
    },
    {
      label: t.chat.fiberLabel,
      value: `${formatMacro(meal.fiber_g)} ${t.profile.gramUnit}`,
    },
  ];

  return (
    <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <div className="flex items-center gap-2">
            <SourceIcon source={meal.input_source} />
            <h3 className="text-base font-semibold leading-snug text-foreground">
              {meal.name}
            </h3>
          </div>
          {meal.portion ? (
            <p className="text-xs text-muted-foreground">{meal.portion}</p>
          ) : null}
        </div>

        <Badge
          className={`h-7 shrink-0 rounded-full border-0 px-3 text-xs font-semibold tabular-nums ${getHealthBadgeClass(meal.health_label)}`}
        >
          {meal.health_score} · {healthLabelText}
        </Badge>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
        {meal.estimated_price_idr > 0 ? (
          <div className="flex items-center gap-1.5">
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
        {macros.map((macro) => (
          <div key={macro.label} className="space-y-0.5 px-1">
            <p className="text-[11px] text-muted-foreground">{macro.label}</p>
            <p className="text-sm font-semibold tabular-nums text-foreground">
              {macro.value}
            </p>
          </div>
        ))}
      </div>

      {meal.note ? (
        <p className="text-sm leading-relaxed text-muted-foreground">
          {meal.note}
        </p>
      ) : null}

      {hasDetails ? (
        <Collapsible open={detailsOpen} onOpenChange={setDetailsOpen}>
          <CollapsibleTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-9 rounded-full px-3 text-xs font-medium text-foreground shadow-none"
              />
            }
          >
            {detailsOpen ? (
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
              {detailsOpen
                ? t.chat.hideDetailsButton
                : t.chat.showDetailsButton}
            </span>
          </CollapsibleTrigger>

          <CollapsibleContent className="pt-2">
            <div className="rounded-2xl bg-muted p-4 space-y-3 text-sm">
              {hasMissing ? (
                <div className="space-y-1.5">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                    {t.chat.missingIngredientsHeading}
                  </p>
                  <ul className="list-disc pl-5 space-y-1 text-foreground">
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
                  <ol className="list-decimal pl-5 space-y-1 text-foreground">
                    {meal.cooking_steps.map((step, idx) => (
                      <li key={`${step}-${idx}`}>{step}</li>
                    ))}
                  </ol>
                </div>
              ) : null}
            </div>
          </CollapsibleContent>
        </Collapsible>
      ) : null}

      <Separator className="bg-border" />

      <div className="space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">
                {t.chat.mealDateSelectLabel}
              </span>
              <Input
                type="date"
                min={todayStr}
                max={maxDateStr}
                value={selectedMealDate}
                disabled={saved || isSaving}
                onChange={(e) => {
                  const next = e.target.value;
                  if (next && next >= todayStr && next <= maxDateStr) {
                    setSelectedMealDate(next);
                  }
                }}
                aria-label={t.chat.mealDateSelectLabel}
                className="h-9 w-auto rounded-full border-border bg-card px-3 text-xs tabular-nums shadow-none"
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
                  className="h-9 min-w-28 rounded-full border-border bg-card px-3 text-xs shadow-none"
                >
                  <SelectValue>{mealTimeLabels[selectedMealTime]}</SelectValue>
                </SelectTrigger>
                <SelectContent className="rounded-2xl border border-border bg-card shadow-none ring-0">
                  <SelectItem value="auto" className="rounded-xl text-xs">
                    {t.meals.mealTimeAuto}
                  </SelectItem>
                  <SelectItem value="breakfast" className="rounded-xl text-xs">
                    {t.meals.mealTimes.breakfast}
                  </SelectItem>
                  <SelectItem value="lunch" className="rounded-xl text-xs">
                    {t.meals.mealTimes.lunch}
                  </SelectItem>
                  <SelectItem value="dinner" className="rounded-xl text-xs">
                    {t.meals.mealTimes.dinner}
                  </SelectItem>
                  <SelectItem value="snack" className="rounded-xl text-xs">
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
    </Card>
  );
}
