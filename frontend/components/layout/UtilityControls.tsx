"use client";

import { Moon, Sun } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";
import { Button } from "@/components/ui/button";

export function UtilityControls() {
  const { language, t, toggleLanguage } = useI18n();
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="flex items-center gap-1.5">
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={toggleLanguage}
        aria-label={t.common.languageLabel}
        className="h-9 min-w-10 rounded-full px-3 text-xs font-semibold tabular-nums shadow-none"
      >
        {language === "id" ? "ID" : "EN"}
      </Button>
      <Button
        type="button"
        variant="outline"
        size="icon"
        onClick={toggleTheme}
        aria-label={t.common.themeLabel}
        title={theme === "dark" ? t.common.themeDark : t.common.themeLight}
        className="size-9 rounded-full shadow-none"
      >
        {theme === "dark" ? (
          <Moon className="size-4 stroke-[1.75]" aria-hidden="true" />
        ) : (
          <Sun className="size-4 stroke-[1.75]" aria-hidden="true" />
        )}
      </Button>
    </div>
  );
}
