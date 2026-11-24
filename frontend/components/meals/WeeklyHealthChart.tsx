"use client";

import { Bar, BarChart, XAxis, YAxis } from "recharts";
import { useI18n } from "@/components/providers/I18nProvider";
import { Card } from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { formatShortChartDate } from "@/lib/date";
import type { WeeklyHealthTrendPoint } from "@/lib/types";

interface WeeklyHealthChartProps {
  points: WeeklyHealthTrendPoint[];
}

export function WeeklyHealthChart({ points }: WeeklyHealthChartProps) {
  const { language, t } = useI18n();

  const chartConfig = {
    score: {
      label: t.chat.healthScoreLabel,
      color: "var(--brand-primary)",
    },
  } satisfies ChartConfig;

  const data = points.map((point) => {
    const { day, date } = formatShortChartDate(point.date, language);
    const hasScore = point.average_health_score !== null;
    const clampedScore = hasScore
      ? Math.max(0, Math.min(100, point.average_health_score ?? 0))
      : 0;
    return {
      key: point.date,
      label: `${day} ${date}`,
      day,
      score: clampedScore,
      hasScore,
      mealCount: point.meal_count,
    };
  });

  return (
    <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-4">
      <div className="space-y-1">
        <h2 className="text-base font-semibold text-foreground">
          {t.history.weeklyChartHeading}
        </h2>
        <p className="text-xs text-muted-foreground">
          {t.history.weeklyChartCaption}
        </p>
      </div>

      <ChartContainer config={chartConfig} className="h-48 w-full">
        <BarChart
          data={data}
          margin={{ top: 8, right: 4, left: -24, bottom: 0 }}
        >
          <XAxis
            dataKey="day"
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            className="text-xs"
          />
          <YAxis
            domain={[0, 100]}
            tickLine={false}
            axisLine={false}
            ticks={[0, 50, 100]}
            className="text-xs tabular-nums"
          />
          <ChartTooltip
            cursor={{ fill: "var(--muted)" }}
            content={<ChartTooltipContent hideIndicator />}
          />
          <Bar
            dataKey="score"
            fill="var(--color-score)"
            radius={8}
            maxBarSize={36}
          />
        </BarChart>
      </ChartContainer>
    </Card>
  );
}
