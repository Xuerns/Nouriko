"use client";

import { useState } from "react";
import { CalendarDays } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { getLocalDateString, parseLocalDate } from "@/lib/date";
import { cn } from "@/lib/utils";

interface DatePickerProps {
  value: string;
  onChange: (nextDate: string) => void;
  minDate?: string;
  maxDate?: string;
  disabled?: boolean;
  ariaLabel: string;
  size?: "default" | "sm";
  className?: string;
}

export function DatePicker({
  value,
  onChange,
  minDate,
  maxDate,
  disabled = false,
  ariaLabel,
  size = "default",
  className,
}: DatePickerProps) {
  const { language, t } = useI18n();
  const [open, setOpen] = useState(false);

  const selectedDate = parseLocalDate(value) ?? undefined;
  const parsedMin = minDate ? parseLocalDate(minDate) : null;
  const parsedMax = maxDate ? parseLocalDate(maxDate) : null;

  const formattedLabel = selectedDate
    ? new Intl.DateTimeFormat(language === "id" ? "id-ID" : "en-US", {
        day: "numeric",
        month: "short",
        year: "numeric",
      }).format(selectedDate)
    : t.common.pickDateLabel;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            type="button"
            variant="outline"
            disabled={disabled}
            aria-label={ariaLabel}
            className={cn(
              "justify-start gap-2 border-border bg-card font-normal tabular-nums text-foreground shadow-none hover:bg-accent",
              size === "sm"
                ? "h-9 rounded-full px-3 text-xs"
                : "h-11 rounded-2xl px-3.5 text-sm",
              className
            )}
          />
        }
      >
        <CalendarDays
          className="size-3.5 shrink-0 stroke-[1.75] text-muted-foreground"
          aria-hidden="true"
        />
        <span>{formattedLabel}</span>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        sideOffset={6}
        className="w-auto rounded-3xl border border-border bg-card p-2 shadow-none ring-0"
      >
        <Calendar
          mode="single"
          selected={selectedDate}
          defaultMonth={selectedDate}
          disabled={(day) => {
            if (parsedMin && day < parsedMin) return true;
            if (parsedMax && day > parsedMax) return true;
            return false;
          }}
          onSelect={(day) => {
            if (!day) return;
            const nextStr = getLocalDateString(day);
            if (minDate && nextStr < minDate) return;
            if (maxDate && nextStr > maxDate) return;
            onChange(nextStr);
            setOpen(false);
          }}
          className="bg-card"
        />
      </PopoverContent>
    </Popover>
  );
}
