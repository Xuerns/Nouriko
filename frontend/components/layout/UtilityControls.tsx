"use client";

import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";

export function UtilityControls() {
  const { language, t, toggleLanguage } = useI18n();
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="flex items-center gap-1.5">
      <button
        type="button"
        onClick={toggleLanguage}
        aria-label={t.common.languageLabel}
        className="rounded-md border border-neutral-300 dark:border-neutral-700 px-2.5 py-1.5 font-mono text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
      >
        {language === "id" ? "ID" : "EN"}
      </button>
      <button
        type="button"
        onClick={toggleTheme}
        aria-label={t.common.themeLabel}
        className="rounded-md border border-neutral-300 dark:border-neutral-700 px-2.5 py-1.5 text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
      >
        {theme === "dark" ? t.common.themeDark : t.common.themeLight}
      </button>
    </div>
  );
}
