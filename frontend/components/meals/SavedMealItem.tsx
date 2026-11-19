"use client";

import { Loader2, Trash2 } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { formatIDR, formatMacro } from "@/components/chat/format";
import type { HealthLabel, SavedMealDTO } from "@/lib/types";

interface SavedMealItemProps {
  meal: SavedMealDTO;
  onDelete?: (mealId: string) => void | Promise<void>;
  isDeleting?: boolean;
}

export function getHealthLabelTextClass(label: HealthLabel): string {
  if (label === "good") {
    return "text-emerald-800 dark:text-emerald-300";
  }
  if (label === "fair") {
    return "text-amber-800 dark:text-amber-300";
  }
  return "text-red-800 dark:text-red-300";
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
    <article className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 space-y-3">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0 space-y-1">
          <h3 className="text-base font-semibold text-neutral-900 dark:text-neutral-100 leading-snug">
            {meal.name}
          </h3>
          <div className="flex flex-wrap items-center gap-2 text-xs text-neutral-600 dark:text-neutral-400">
            <span className="rounded border border-neutral-300 dark:border-neutral-700 px-2 py-0.5 font-medium text-neutral-800 dark:text-neutral-200">
              {mealTimeLabel}
            </span>
            <span className="rounded border border-neutral-300 dark:border-neutral-700 px-2 py-0.5 font-mono text-neutral-700 dark:text-neutral-300">
              {sourceLabel}
            </span>
          </div>
        </div>

        {onDelete ? (
          <button
            type="button"
            disabled={isDeleting}
            onClick={() => void onDelete(meal.id)}
            aria-label={`${t.meals.deleteButton} ${meal.name}`}
            className="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-neutral-300 dark:border-neutral-700 px-2.5 py-1.5 text-xs font-medium text-red-700 dark:text-red-400 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
          >
            {isDeleting ? (
              <Loader2
                className="h-3.5 w-3.5 shrink-0 animate-spin"
                aria-hidden="true"
              />
            ) : (
              <Trash2 className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            )}
            <span>
              {isDeleting ? t.meals.deletingButton : t.meals.deleteButton}
            </span>
          </button>
        ) : null}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 text-sm">
        <div className="flex items-center gap-2">
          <span className="text-neutral-600 dark:text-neutral-400">
            {t.chat.healthScoreLabel}:
          </span>
          <span className="font-mono font-semibold text-neutral-900 dark:text-neutral-100">
            {meal.health_score}/100
          </span>
          <span
            className={`font-medium ${getHealthLabelTextClass(meal.health_label)}`}
          >
            ({healthLabelText})
          </span>
        </div>

        {meal.price_idr !== null && meal.price_idr > 0 ? (
          <div className="flex items-center gap-1.5">
            <span className="text-neutral-600 dark:text-neutral-400">
              {t.chat.priceLabel}:
            </span>
            <span className="font-mono font-semibold text-neutral-900 dark:text-neutral-100">
              {formatIDR(meal.price_idr)}
            </span>
          </div>
        ) : null}
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-5 rounded-md border border-neutral-300 dark:border-neutral-700 divide-y sm:divide-y-0 sm:divide-x divide-neutral-300 dark:divide-neutral-700">
        {macros.map((macro) => (
          <div key={macro.label} className="p-2.5">
            <p className="text-xs text-neutral-500 dark:text-neutral-400">
              {macro.label}
            </p>
            <p className="mt-0.5 font-mono text-sm font-medium text-neutral-900 dark:text-neutral-100">
              {macro.value}
            </p>
          </div>
        ))}
      </div>
    </article>
  );
}
