"use client";

import { Moon, Sun } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";

export function UtilityControls() {
  const { language, t, toggleLanguage } = useI18n();
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="flex items-center gap-2">
      <button
        type="button"
        onClick={toggleLanguage}
        aria-label={t.common.languageLabel}
        className="inline-flex h-9 items-center justify-center rounded-full border border-border px-3 text-xs font-semibold tracking-wider text-foreground shadow-none transition-colors duration-150 hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)]"
      >
        {language === "id" ? "ID" : "EN"}
      </button>
      <button
        type="button"
        onClick={toggleTheme}
        aria-label={t.common.themeLabel}
        className="inline-flex h-9 w-9 items-center justify-center rounded-full border border-border text-foreground shadow-none transition-colors duration-150 hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)]"
      >
        {theme === "dark" ? (
          <Sun className="h-4 w-4" strokeWidth={1.75} />
        ) : (
          <Moon className="h-4 w-4" strokeWidth={1.75} />
        )}
      </button>
    </div>
  );
}
