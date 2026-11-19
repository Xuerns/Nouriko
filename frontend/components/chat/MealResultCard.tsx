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
import type {
  HealthLabel,
  InputSource,
  MealOption,
  MealTime,
} from "@/lib/types";

interface MealResultCardProps {
  meal: MealOption;
  onSave?: (meal: MealOption, mealTime?: MealTime) => void | Promise<void>;
  saved?: boolean;
  isSaving?: boolean;
  saveError?: string | null;
}

function SourceIcon({ source }: { source: InputSource }) {
  const iconClass = "h-4 w-4 shrink-0 text-neutral-600 dark:text-neutral-400";
  if (source === "budget") {
    return <ShoppingBasket className={iconClass} aria-hidden="true" />;
  }
  if (source === "photo") {
    return <Camera className={iconClass} aria-hidden="true" />;
  }
  return <Utensils className={iconClass} aria-hidden="true" />;
}

function getHealthLabelTextClass(label: HealthLabel): string {
  if (label === "good") {
    return "text-emerald-800 dark:text-emerald-300";
  }
  if (label === "fair") {
    return "text-amber-800 dark:text-amber-300";
  }
  return "text-red-800 dark:text-red-300";
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
  const [selectedMealTime, setSelectedMealTime] = useState<"" | MealTime>("");

  const hasMissing =
    Array.isArray(meal.missing_ingredients) &&
    meal.missing_ingredients.length > 0;
  const hasSteps =
    Array.isArray(meal.cooking_steps) && meal.cooking_steps.length > 0;
  const hasDetails = hasMissing || hasSteps;

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
        <div className="min-w-0 space-y-0.5">
          <div className="flex items-center gap-2">
            <SourceIcon source={meal.input_source} />
            <h3 className="text-base font-semibold text-neutral-900 dark:text-neutral-100 leading-snug">
              {meal.name}
            </h3>
          </div>
          {meal.portion ? (
            <p className="text-xs text-neutral-600 dark:text-neutral-400">
              {meal.portion}
            </p>
          ) : null}
        </div>

        <span className="shrink-0 rounded border border-neutral-300 dark:border-neutral-700 px-2 py-0.5 font-mono text-xs text-neutral-700 dark:text-neutral-300">
          {t.chat.estimatedTag}
        </span>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1.5 text-sm">
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

        {meal.estimated_price_idr > 0 ? (
          <div className="flex items-center gap-1.5">
            <span className="text-neutral-600 dark:text-neutral-400">
              {t.chat.priceLabel}:
            </span>
            <span className="font-mono font-semibold text-neutral-900 dark:text-neutral-100">
              {formatIDR(meal.estimated_price_idr)}
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

      {meal.note ? (
        <p className="text-sm text-neutral-700 dark:text-neutral-300 leading-relaxed">
          {meal.note}
        </p>
      ) : null}

      {hasDetails ? (
        <div className="space-y-2.5">
          <button
            type="button"
            onClick={() => setDetailsOpen((prev) => !prev)}
            aria-expanded={detailsOpen}
            className="inline-flex items-center gap-1.5 text-xs font-medium text-neutral-700 dark:text-neutral-300 underline underline-offset-4 transition-colors duration-150 hover:text-neutral-900 dark:hover:text-neutral-100"
          >
            {detailsOpen ? (
              <ChevronUp className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            ) : (
              <ChevronDown
                className="h-3.5 w-3.5 shrink-0"
                aria-hidden="true"
              />
            )}
            <span>
              {detailsOpen
                ? t.chat.hideDetailsButton
                : t.chat.showDetailsButton}
            </span>
          </button>

          {detailsOpen ? (
            <div className="rounded-md border border-neutral-300 dark:border-neutral-700 p-3 space-y-3 text-sm">
              {hasMissing ? (
                <div className="space-y-1">
                  <p className="font-mono text-xs uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
                    {t.chat.missingIngredientsHeading}
                  </p>
                  <ul className="list-disc pl-5 space-y-0.5 text-neutral-800 dark:text-neutral-200">
                    {meal.missing_ingredients.map((item, idx) => (
                      <li key={`${item}-${idx}`}>{item}</li>
                    ))}
                  </ul>
                </div>
              ) : null}

              {hasSteps ? (
                <div className="space-y-1">
                  <p className="font-mono text-xs uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
                    {t.chat.cookingStepsHeading}
                  </p>
                  <ol className="list-decimal pl-5 space-y-1 text-neutral-800 dark:text-neutral-200">
                    {meal.cooking_steps.map((step, idx) => (
                      <li key={`${step}-${idx}`}>{step}</li>
                    ))}
                  </ol>
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      ) : null}

      <div className="space-y-2 border-t border-neutral-300 dark:border-neutral-700 pt-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <label className="text-xs text-neutral-600 dark:text-neutral-400">
              <span>{t.chat.mealTimeSelectLabel}</span>
              <select
                value={selectedMealTime}
                disabled={saved || isSaving}
                onChange={(e) =>
                  setSelectedMealTime(e.target.value as "" | MealTime)
                }
                aria-label={t.chat.mealTimeSelectLabel}
                className="ml-2 rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-2 py-1 text-xs text-neutral-900 dark:text-neutral-100 disabled:opacity-50"
              >
                <option
                  value=""
                  className="bg-white text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100"
                >
                  {t.meals.mealTimeAuto}
                </option>
                <option
                  value="breakfast"
                  className="bg-white text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100"
                >
                  {t.meals.mealTimes.breakfast}
                </option>
                <option
                  value="lunch"
                  className="bg-white text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100"
                >
                  {t.meals.mealTimes.lunch}
                </option>
                <option
                  value="dinner"
                  className="bg-white text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100"
                >
                  {t.meals.mealTimes.dinner}
                </option>
                <option
                  value="snack"
                  className="bg-white text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100"
                >
                  {t.meals.mealTimes.snack}
                </option>
              </select>
            </label>
          </div>

          <button
            type="button"
            disabled={saved || isSaving}
            onClick={() =>
              void onSave?.(
                meal,
                selectedMealTime === "" ? undefined : selectedMealTime
              )
            }
            aria-pressed={saved}
            className="inline-flex items-center gap-1.5 rounded-md border border-neutral-300 dark:border-neutral-700 px-3 py-1.5 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-60"
          >
            {isSaving ? (
              <Loader2
                className="h-3.5 w-3.5 shrink-0 animate-spin"
                aria-hidden="true"
              />
            ) : saved ? (
              <Check className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            ) : (
              <Bookmark className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            )}
            <span>
              {isSaving
                ? t.chat.savingMealButton
                : saved
                  ? t.chat.savedMealButton
                  : t.chat.saveMealButton}
            </span>
          </button>
        </div>

        {saveError ? (
          <p role="alert" className="text-xs text-red-700 dark:text-red-400">
            {saveError}
          </p>
        ) : null}
      </div>
    </article>
  );
}
