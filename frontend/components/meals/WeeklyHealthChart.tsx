"use client";

import { useId } from "react";
import { useI18n } from "@/components/providers/I18nProvider";
import { Card } from "@/components/ui/card";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { formatShortChartDate } from "@/lib/date";
import type { WeeklyHealthTrendPoint } from "@/lib/types";

interface WeeklyHealthChartProps {
  points: WeeklyHealthTrendPoint[];
}

const BODY_TOP_Y = 6;
const BODY_BOTTOM_Y = 235;
const BODY_HEIGHT = BODY_BOTTOM_Y - BODY_TOP_Y;

const HUMAN_SILHOUETTE_PATH =
  "M 50,6 " +
  "C 56,6 60,11 60,18 " +
  "C 60,24 59,29 56.5,33 " +
  "C 55.5,35 55,37 55,40 " +
  "C 55,44 59,46 66,49 " +
  "C 73,52 77,57 78,66 " +
  "C 79,76 80,86 82,96 " +
  "C 84,106 86,113 88,120 " +
  "C 90,125 90,132 87.5,135 " +
  "C 86,136.5 84,135 83.5,132 " +
  "C 83,129 82.5,127 81.5,127 " +
  "C 81,129 80,130 79,129 " +
  "C 78,128 79,124 80,120 " +
  "C 78,112 75,102 72,92 " +
  "C 70,85 69,79 68,75 " +
  "C 67,85 65.5,95 65,104 " +
  "C 65,113 68,120 68,132 " +
  "C 68,146 66,158 64.5,170 " +
  "C 64,180 65,192 63,204 " +
  "C 61.5,212 61,218 61.5,223 " +
  "C 62,227 65,230 65.5,233 " +
  "C 65.5,234.5 63,235 59,235 " +
  "L 55.5,235 " +
  "C 53.5,235 53,233.5 53.5,231 " +
  "C 54.5,224 55,216 54.5,205 " +
  "C 54,194 53.5,182 54,170 " +
  "C 54.5,156 53,142 51.5,130 " +
  "C 51,128 50.5,127.5 50,127.5 " +
  "C 49.5,127.5 49,128 48.5,130 " +
  "C 47,142 45.5,156 46,170 " +
  "C 46.5,182 46,194 45.5,205 " +
  "C 45,216 45.5,224 46.5,231 " +
  "C 47,233.5 46.5,235 44.5,235 " +
  "L 41,235 " +
  "C 37,235 34.5,234.5 34.5,233 " +
  "C 35,230 38,227 38.5,223 " +
  "C 39,218 38.5,212 37,204 " +
  "C 35,192 36,180 35.5,170 " +
  "C 34,158 32,146 32,132 " +
  "C 32,120 35,113 35,104 " +
  "C 34.5,95 33,85 32,75 " +
  "C 31,79 30,85 28,92 " +
  "C 25,102 22,112 20,120 " +
  "C 21,124 22,128 21,129 " +
  "C 20,130 19,129 18.5,127 " +
  "C 17.5,127 17,129 16.5,132 " +
  "C 16,135 14,136.5 12.5,135 " +
  "C 10,132 10,125 12,120 " +
  "C 14,113 16,106 18,96 " +
  "C 20,86 21,76 22,66 " +
  "C 23,57 27,52 34,49 " +
  "C 41,46 45,44 45,40 " +
  "C 45,37 44.5,35 43.5,33 " +
  "C 41,29 40,24 40,18 " +
  "C 40,11 44,6 50,6 Z";

function getScoreFillColor(score: number): string {
  if (score >= 70) {
    return "var(--brand-primary)";
  }
  if (score >= 40) {
    return "var(--macro-carbs)";
  }
  return "var(--macro-fat)";
}

export function WeeklyHealthChart({ points }: WeeklyHealthChartProps) {
  const { language, t } = useI18n();
  const baseId = useId();

  const data = points.map((point) => {
    const { day, date } = formatShortChartDate(point.date, language);
    const hasScore = point.average_health_score !== null;
    const clampedScore = hasScore
      ? Math.max(0, Math.min(100, Math.round(point.average_health_score ?? 0)))
      : 0;
    return {
      key: point.date,
      label: `${day} ${date}`,
      day,
      date,
      score: clampedScore,
      hasScore,
      mealCount: point.meal_count,
    };
  });

  return (
    <Card className="gap-5 rounded-3xl border border-border bg-card p-5 shadow-none ring-0 sm:p-6">
      <div className="space-y-1">
        <h2 className="text-[18px] font-semibold leading-[1.3] tracking-[-0.01em] text-foreground sm:text-[20px]">
          {t.history.weeklyChartHeading}
        </h2>
        <p className="text-[13px] leading-[1.5] text-muted-foreground">
          {t.history.weeklyChartCaption}
        </p>
      </div>

      <div className="grid grid-cols-7 gap-1 sm:gap-2">
        {data.map((item, index) => {
          const clipId = `${baseId}-body-${index}`;
          const fillHeight = item.hasScore
            ? (item.score / 100) * BODY_HEIGHT
            : 0;
          const fillY = BODY_BOTTOM_Y - fillHeight;
          const fillColor = getScoreFillColor(item.score);
          const tooltipText = item.hasScore
            ? `${item.label} · ${item.score}/100 · ${item.mealCount} ${t.meals.mealCountUnit}`
            : `${item.label} · ${t.history.noDataLabel}`;

          return (
            <Tooltip key={item.key}>
              <TooltipTrigger
                render={
                  <div
                    tabIndex={0}
                    className="flex flex-col items-center gap-2 rounded-2xl px-1 py-2.5 transition-colors duration-150 hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)] sm:px-2"
                  />
                }
              >
                <span
                  className={`text-xs font-semibold tabular-nums ${
                    item.hasScore ? "text-foreground" : "text-muted-foreground"
                  }`}
                >
                  {item.hasScore ? item.score : "-"}
                </span>

                <svg
                  viewBox="0 0 100 240"
                  role="img"
                  aria-label={tooltipText}
                  className="h-auto w-full max-w-[54px] overflow-visible"
                >
                  <defs>
                    <clipPath id={clipId}>
                      <rect
                        x="0"
                        y={fillY}
                        width="100"
                        height={fillHeight}
                        className="transition-all duration-300"
                      />
                    </clipPath>
                  </defs>

                  <path
                    d={HUMAN_SILHOUETTE_PATH}
                    fill="var(--muted)"
                    stroke="var(--border)"
                    strokeWidth="1.5"
                  />

                  {item.hasScore ? (
                    <path
                      d={HUMAN_SILHOUETTE_PATH}
                      fill={fillColor}
                      clipPath={`url(#${clipId})`}
                    />
                  ) : null}
                </svg>

                <p className="text-center text-xs font-medium leading-tight text-foreground">
                  {item.day}
                </p>
              </TooltipTrigger>
              <TooltipContent className="rounded-xl shadow-none">
                {tooltipText}
              </TooltipContent>
            </Tooltip>
          );
        })}
      </div>
    </Card>
  );
}
