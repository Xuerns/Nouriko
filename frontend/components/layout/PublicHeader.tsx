"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Languages, Menu, Moon, Sun } from "lucide-react";
import { BrandLogo } from "@/components/layout/BrandLogo";
import { UtilityControls } from "@/components/layout/UtilityControls";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export function PublicHeader() {
  const pathname = usePathname();
  const { isAuthenticated, isLoading, profile } = useAuth();
  const { language, t, toggleLanguage } = useI18n();
  const { toggleTheme } = useTheme();

  const appHref = profile?.onboarding_completed ? "/chat" : "/onboarding";
  const appLabel = profile?.onboarding_completed
    ? t.nav.chat
    : t.landing.ctaContinueOnboarding;

  return (
    <header className="sticky top-0 z-30 w-full border-b border-border bg-background shadow-none">
      <div className="mx-auto flex h-16 w-full max-w-5xl items-center justify-between gap-4 px-4 sm:px-6">
        <BrandLogo href="/" isCurrentPage={pathname === "/"} />

        <div className="hidden md:flex items-center gap-2">
          {isLoading ? (
            <div className="h-9 w-28" aria-hidden="true" />
          ) : isAuthenticated ? (
            <Button
              size="sm"
              nativeButton={false}
              render={<Link href={appHref} />}
              className="min-h-9 rounded-full bg-primary px-4 text-sm font-medium text-primary-foreground shadow-none transition-colors duration-150 hover:bg-[var(--brand-primary-hover)] active:translate-y-0"
            >
              {appLabel}
            </Button>
          ) : (
            <>
              <Button
                variant="ghost"
                size="sm"
                nativeButton={false}
                render={<Link href="/login" />}
                className="min-h-9 rounded-full px-3.5 text-sm font-medium text-foreground shadow-none transition-colors duration-150 hover:bg-accent active:translate-y-0"
              >
                {t.nav.login}
              </Button>
              <Button
                size="sm"
                nativeButton={false}
                render={<Link href="/register" />}
                className="min-h-9 rounded-full bg-primary px-4 text-sm font-medium text-primary-foreground shadow-none transition-colors duration-150 hover:bg-[var(--brand-primary-hover)] active:translate-y-0"
              >
                {t.nav.register}
              </Button>
            </>
          )}

          <span
            aria-hidden="true"
            className="mx-1 h-4 w-px shrink-0 bg-border"
          />

          <UtilityControls />
        </div>

        <div className="flex md:hidden items-center">
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  aria-label={t.nav.menuLabel}
                  className="size-11 rounded-full border-border bg-card text-foreground shadow-none transition-colors duration-150 hover:bg-accent active:translate-y-0"
                />
              }
            >
              <Menu className="size-4" strokeWidth={1.75} aria-hidden="true" />
            </DropdownMenuTrigger>

            <DropdownMenuContent
              align="end"
              sideOffset={8}
              className="w-60 rounded-2xl border border-border bg-card p-1.5 text-card-foreground shadow-none ring-0"
            >
              {!isLoading && isAuthenticated ? (
                <DropdownMenuItem
                  render={<Link href={appHref} />}
                  className="min-h-11 rounded-xl bg-primary px-3 text-sm font-medium text-primary-foreground focus:bg-[var(--brand-primary-hover)] focus:text-primary-foreground"
                >
                  {appLabel}
                </DropdownMenuItem>
              ) : (
                <>
                  <DropdownMenuItem
                    render={<Link href="/login" />}
                    className="min-h-11 rounded-xl px-3 text-sm font-medium text-foreground"
                  >
                    {t.nav.login}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    render={<Link href="/register" />}
                    className="min-h-11 rounded-xl px-3 text-sm font-medium text-[var(--brand-primary)]"
                  >
                    {t.nav.register}
                  </DropdownMenuItem>
                </>
              )}

              <DropdownMenuSeparator className="my-1.5 bg-border" />

              <DropdownMenuItem
                render={<button type="button" />}
                nativeButton
                closeOnClick={false}
                onClick={toggleLanguage}
                className="min-h-11 w-full justify-between rounded-xl px-3 text-sm font-medium text-foreground"
              >
                <span className="inline-flex items-center gap-2.5">
                  <Languages
                    className="size-4 text-muted-foreground"
                    strokeWidth={1.75}
                    aria-hidden="true"
                  />
                  <span>{t.common.languageLabel}</span>
                </span>
                <span className="rounded-full bg-[var(--bg-subtle)] px-2 py-0.5 font-mono text-[11px] font-semibold text-foreground">
                  {language === "id" ? "ID" : "EN"}
                </span>
              </DropdownMenuItem>

              <DropdownMenuItem
                render={<button type="button" />}
                nativeButton
                closeOnClick={false}
                onClick={(event) => toggleTheme(event)}
                className="min-h-11 w-full justify-between rounded-xl px-3 text-sm font-medium text-foreground"
              >
                <span className="inline-flex items-center gap-2.5">
                  <Sun
                    className="size-4 text-muted-foreground dark:hidden"
                    strokeWidth={1.75}
                    aria-hidden="true"
                  />
                  <Moon
                    className="hidden size-4 text-muted-foreground dark:block"
                    strokeWidth={1.75}
                    aria-hidden="true"
                  />
                  <span>{t.common.themeLabel}</span>
                </span>
                <span className="text-xs text-muted-foreground dark:hidden">
                  {t.common.themeLight}
                </span>
                <span className="hidden text-xs text-muted-foreground dark:inline">
                  {t.common.themeDark}
                </span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      <div
        aria-hidden="true"
        className="nouriko-scroll-progress pointer-events-none absolute inset-x-0 bottom-0 hidden h-0.5 bg-[var(--brand-primary)]"
      />
    </header>
  );
}
