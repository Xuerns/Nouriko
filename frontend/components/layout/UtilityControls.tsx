"use client";

import { Moon, Sun } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { Language } from "@/lib/types";

const LANGUAGES: readonly { code: Language; label: string }[] = [
  { code: "id", label: "ID" },
  { code: "en", label: "EN" },
];

export function UtilityControls() {
  const { language, t, setLanguage } = useI18n();
  const { toggleTheme } = useTheme();

  return (
    <div className="flex items-center gap-2">
      <div
        role="group"
        aria-label={t.common.languageLabel}
        className="inline-flex h-10 sm:h-9 items-center rounded-full border border-border bg-[var(--bg-subtle)] p-0.5"
      >
        {LANGUAGES.map((item) => {
          const isActive = language === item.code;
          return (
            <button
              key={item.code}
              type="button"
              aria-pressed={isActive}
              onClick={() => setLanguage(item.code)}
              className={`inline-flex h-full min-w-9 sm:min-w-8 cursor-pointer items-center justify-center rounded-full px-2.5 sm:px-2 font-mono text-xs sm:text-[11px] tracking-wide transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring ${
                isActive
                  ? "bg-card font-semibold text-foreground"
                  : "font-medium text-muted-foreground hover:text-foreground"
              }`}
            >
              {item.label}
            </button>
          );
        })}
      </div>

      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={(event) => toggleTheme(event)}
              className="size-10 sm:size-9 cursor-pointer rounded-full border-border bg-[var(--bg-subtle)] text-foreground shadow-none transition-colors duration-150 hover:bg-accent dark:border-border dark:bg-[var(--bg-subtle)] dark:hover:bg-accent active:translate-y-0"
            />
          }
        >
          <Sun
            className="size-4 dark:hidden"
            strokeWidth={1.75}
            aria-hidden="true"
          />
          <span className="sr-only dark:hidden">
            {`${t.common.themeLabel}: ${t.common.themeLight}`}
          </span>
          <Moon
            className="hidden size-4 dark:block"
            strokeWidth={1.75}
            aria-hidden="true"
          />
          <span className="sr-only hidden dark:inline">
            {`${t.common.themeLabel}: ${t.common.themeDark}`}
          </span>
        </TooltipTrigger>
        <TooltipContent className="rounded-xl shadow-none">
          {t.common.themeLabel}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
