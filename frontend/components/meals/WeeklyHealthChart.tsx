"use client";

import { useI18n } from "@/components/providers/I18nProvider";
import { formatShortChartDate } from "@/lib/date";
import type { WeeklyHealthTrendPoint } from "@/lib/types";

interface WeeklyHealthChartProps {
  points: WeeklyHealthTrendPoint[];
}

export function WeeklyHealthChart({ points }: WeeklyHealthChartProps) {
  const { language, t } = useI18n();

  return (
    <figure className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 sm:p-5 space-y-4">
      <div className="space-y-1">
        <h2 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
          {t.history.weeklyChartHeading}
        </h2>
        <figcaption className="text-xs text-neutral-600 dark:text-neutral-400">
          {t.history.weeklyChartCaption}
        </figcaption>
      </div>

      <div
        className="grid grid-cols-7 gap-2 sm:gap-3"
        role="img"
        aria-label={t.history.weeklyChartHeading}
      >
        {points.map((point) => {
          const { day, date } = formatShortChartDate(point.date, language);
          const hasScore = point.average_health_score !== null;
          const clampedScore = hasScore
            ? Math.max(0, Math.min(100, point.average_health_score ?? 0))
            : 0;
          const ariaLabel = hasScore
            ? `${point.date}: ${clampedScore}/100 (${point.meal_count} ${t.meals.mealCountUnit})`
            : `${point.date}: ${t.history.noDataLabel}`;

          return (
            <div
              key={point.date}
              aria-label={ariaLabel}
              className="flex flex-col items-center gap-1.5 min-w-0"
            >
              <span className="font-mono text-xs font-medium text-neutral-900 dark:text-neutral-100">
                {hasScore ? clampedScore : "-"}
              </span>

              <div className="flex h-28 w-full max-w-10 items-end rounded-sm border border-neutral-300 dark:border-neutral-700 p-0.5">
                {hasScore ? (
                  <div
                    className="w-full rounded-sm bg-neutral-900 dark:bg-neutral-100"
                    style={{
                      height: `${Math.max(clampedScore, 4)}%`,
                    }}
                  />
                ) : null}
              </div>

              <div className="text-center leading-tight">
                <p className="text-xs font-medium text-neutral-800 dark:text-neutral-200 truncate">
                  {day}
                </p>
                <p className="font-mono text-xs text-neutral-500 dark:text-neutral-400">
                  {date}
                </p>
              </div>
            </div>
          );
        })}
      </div>
    </figure>
  );
}
