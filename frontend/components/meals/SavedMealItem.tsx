"use client";

import { Loader2, Trash2 } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { formatIDR, formatMacro } from "@/components/chat/format";
import { getHealthBadgeClass } from "@/components/chat/MealResultCard";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import type { HealthLabel, SavedMealDTO } from "@/lib/types";

interface SavedMealItemProps {
  meal: SavedMealDTO;
  onDelete?: (mealId: string) => void | Promise<void>;
  isDeleting?: boolean;
}

export function getHealthLabelTextClass(label: HealthLabel): string {
  if (label === "good") {
    return "text-[var(--score-good-text)]";
  }
  if (label === "fair") {
    return "text-[var(--score-fair-text)]";
  }
  return "text-[var(--score-low-text)]";
}

export function SavedMealItem({
  meal,
  onDelete,
  isDeleting = false,
}: SavedMealItemProps) {
  const { t } = useI18n();

  const mealTimeLabel = Object.hasOwn(t.meals.mealTimes, meal.meal_time)
    ? t.meals.mealTimes[meal.meal_time]
    : meal.meal_time;

  const sourceLabel = Object.hasOwn(t.meals.inputSources, meal.input_source)
    ? t.meals.inputSources[meal.input_source]
    : meal.input_source;

  const healthLabelText = Object.hasOwn(t.chat.healthLabels, meal.health_label)
    ? t.chat.healthLabels[meal.health_label]
    : t.chat.healthLabels.fair;

  const macros = [
    {
      label: t.chat.caloriesLabel,
      value: `${Math.round(meal.calories)}`,
      unit: t.profile.kcalUnit,
    },
    {
      label: t.chat.proteinLabel,
      value: formatMacro(meal.protein_g),
      unit: t.profile.gramUnit,
    },
    {
      label: t.chat.carbsLabel,
      value: formatMacro(meal.carbs_g),
      unit: t.profile.gramUnit,
    },
    {
      label: t.chat.fatLabel,
      value: formatMacro(meal.fat_g),
      unit: t.profile.gramUnit,
    },
    {
      label: t.chat.fiberLabel,
      value: formatMacro(meal.fiber_g),
      unit: t.profile.gramUnit,
    },
  ];

  return (
    <Card className="flex flex-col justify-between gap-3 rounded-3xl border border-border bg-card p-4 shadow-none ring-0">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 space-y-0.5">
          <h3 className="text-sm font-semibold leading-snug text-foreground line-clamp-2">
            {meal.name}
          </h3>
          <p className="text-xs text-muted-foreground">
            {mealTimeLabel} · {sourceLabel}
          </p>
        </div>

        {onDelete ? (
          <AlertDialog>
            <AlertDialogTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  disabled={isDeleting}
                  aria-label={`${t.meals.deleteButton} ${meal.name}`}
                  title={
                    isDeleting ? t.meals.deletingButton : t.meals.deleteButton
                  }
                  className="size-9 shrink-0 rounded-full text-[var(--error-text)] hover:bg-[var(--error-bg)] hover:text-[var(--error-text)] shadow-none"
                />
              }
            >
              {isDeleting ? (
                <Loader2
                  className="size-4 shrink-0 animate-spin stroke-[1.75]"
                  aria-hidden="true"
                />
              ) : (
                <Trash2
                  className="size-4 shrink-0 stroke-[1.75]"
                  aria-hidden="true"
                />
              )}
            </AlertDialogTrigger>
            <AlertDialogContent className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0">
              <AlertDialogHeader>
                <AlertDialogTitle>{t.meals.deleteButton}</AlertDialogTitle>
                <AlertDialogDescription>{meal.name}</AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter className="mx-0 mb-0 rounded-none border-0 bg-transparent p-0 pt-2">
                <AlertDialogCancel className="h-10 rounded-full px-5 shadow-none">
                  {t.common.cancel}
                </AlertDialogCancel>
                <AlertDialogAction
                  variant="destructive"
                  onClick={() => void onDelete(meal.id)}
                  className="h-10 rounded-full bg-[var(--error-bg)] px-5 text-[var(--error-text)] hover:bg-[var(--error-bg)]/80 shadow-none"
                >
                  {t.meals.deleteButton}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        ) : null}
      </div>

      <div className="space-y-2.5">
        <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
          <Badge
            className={`h-6 rounded-full border-0 px-2.5 text-xs font-semibold tabular-nums ${getHealthBadgeClass(meal.health_label)}`}
          >
            {meal.health_score} · {healthLabelText}
          </Badge>

          {meal.price_idr !== null && meal.price_idr > 0 ? (
            <span className="font-semibold tabular-nums text-[var(--brand-accent)]">
              {formatIDR(meal.price_idr)}
            </span>
          ) : null}
        </div>

        <div className="grid grid-cols-5 gap-1 rounded-2xl bg-muted p-2">
          {macros.map((macro) => (
            <div key={macro.label} className="px-1 py-0.5 text-center">
              <p className="truncate text-[10px] text-muted-foreground">
                {macro.label}
              </p>
              <p className="mt-0.5 text-[11px] font-semibold tabular-nums text-foreground whitespace-nowrap">
                {macro.value}
                <span className="text-[10px] font-normal text-muted-foreground">
                  {" "}
                  {macro.unit}
                </span>
              </p>
            </div>
          ))}
        </div>
      </div>
    </Card>
  );
}
