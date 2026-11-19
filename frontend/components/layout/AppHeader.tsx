"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";

export function AppHeader() {
  const pathname = usePathname();
  const { isAuthenticated, isLoading, logout } = useAuth();
  const { language, t, toggleLanguage } = useI18n();
  const { theme, toggleTheme } = useTheme();

  const navLinkClass = (href: string) =>
    `rounded-md border px-2.5 py-1.5 text-xs font-medium transition-colors duration-150 ${
      pathname === href
        ? "border-neutral-900 dark:border-neutral-100 text-neutral-900 dark:text-neutral-100"
        : "border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300 hover:bg-neutral-100 dark:hover:bg-neutral-900"
    }`;

  return (
    <header className="w-full border-b border-neutral-300 dark:border-neutral-700">
      <div className="mx-auto flex w-full max-w-4xl flex-wrap items-center justify-between gap-3 px-4 py-3">
        <Link
          href="/"
          className="text-base font-semibold tracking-tight text-neutral-900 dark:text-neutral-100"
        >
          {t.common.brand}
        </Link>

        <div className="flex flex-wrap items-center gap-2">
          <nav
            aria-label="Primary"
            className="flex flex-wrap items-center gap-2"
          >
            {isLoading ? null : isAuthenticated ? (
              <>
                <Link href="/chat" className={navLinkClass("/chat")}>
                  {t.nav.chat}
                </Link>
                <Link href="/profile" className={navLinkClass("/profile")}>
                  {t.nav.profile}
                </Link>
                <button
                  type="button"
                  onClick={() => void logout()}
                  className="rounded-md border border-neutral-300 dark:border-neutral-700 px-2.5 py-1.5 text-xs font-medium text-neutral-700 dark:text-neutral-300 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
                >
                  {t.nav.logout}
                </button>
              </>
            ) : (
              <>
                <Link href="/login" className={navLinkClass("/login")}>
                  {t.nav.login}
                </Link>
                <Link href="/register" className={navLinkClass("/register")}>
                  {t.nav.register}
                </Link>
              </>
            )}
          </nav>

          <div className="flex items-center gap-1.5 border-l border-neutral-300 dark:border-neutral-700 pl-2">
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
        </div>
      </div>
    </header>
  );
}
