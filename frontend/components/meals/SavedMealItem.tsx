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
    <article className="flex flex-col justify-between gap-2.5 rounded-lg border border-neutral-300 dark:border-neutral-700 p-3.5">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 space-y-0.5">
          <h3 className="text-sm font-semibold text-neutral-900 dark:text-neutral-100 leading-snug line-clamp-2">
            {meal.name}
          </h3>
          <p className="font-mono text-[11px] text-neutral-500 dark:text-neutral-400">
            {mealTimeLabel} / {sourceLabel}
          </p>
        </div>

        {onDelete ? (
          <button
            type="button"
            disabled={isDeleting}
            onClick={() => void onDelete(meal.id)}
            aria-label={`${t.meals.deleteButton} ${meal.name}`}
            title={isDeleting ? t.meals.deletingButton : t.meals.deleteButton}
            className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-neutral-300 dark:border-neutral-700 text-red-700 dark:text-red-400 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
          >
            {isDeleting ? (
              <Loader2
                className="h-3.5 w-3.5 shrink-0 animate-spin"
                aria-hidden="true"
              />
            ) : (
              <Trash2 className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            )}
          </button>
        ) : null}
      </div>

      <div className="space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-0.5 text-xs">
          <div className="flex items-center gap-1.5">
            <span className="text-neutral-500 dark:text-neutral-400">
              {t.chat.healthScoreLabel}
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
            <span className="font-mono font-semibold text-neutral-900 dark:text-neutral-100">
              {formatIDR(meal.price_idr)}
            </span>
          ) : null}
        </div>

        <div className="grid grid-cols-5 rounded-md border border-neutral-200 dark:border-neutral-800 divide-x divide-neutral-200 dark:divide-neutral-800">
          {macros.map((macro) => (
            <div key={macro.label} className="px-1.5 py-1.5 text-center">
              <p className="truncate text-[10px] text-neutral-500 dark:text-neutral-400">
                {macro.label}
              </p>
              <p className="mt-0.5 font-mono text-[11px] font-medium text-neutral-900 dark:text-neutral-100 whitespace-nowrap">
                {macro.value}
                <span className="text-[10px] text-neutral-500 dark:text-neutral-400">
                  {" "}
                  {macro.unit}
                </span>
              </p>
            </div>
          ))}
        </div>
      </div>
    </article>
  );
}
