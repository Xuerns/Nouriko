"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { BarChart3, History, MessageSquare } from "lucide-react";
import { AccountMenu } from "@/components/layout/AccountMenu";
import { BrandLogo } from "@/components/layout/BrandLogo";
import { NourikoMark } from "@/components/layout/NourikoMark";
import { useI18n } from "@/components/providers/I18nProvider";

interface AppShellProps {
  children: ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  const pathname = usePathname();
  const { t } = useI18n();

  const isChatRoute = pathname === "/chat";
  const isSummaryRoute = pathname === "/summary";
  const maxWidthClass = isChatRoute
    ? "max-w-2xl"
    : isSummaryRoute
      ? "max-w-5xl"
      : "max-w-3xl";

  const navItems = [
    { href: "/chat", label: t.nav.chat, icon: MessageSquare },
    { href: "/summary", label: t.nav.summary, icon: BarChart3 },
    { href: "/history", label: t.nav.history, icon: History },
  ] as const;

  return (
    <div
      className={
        isChatRoute
          ? "flex h-dvh flex-col overflow-hidden bg-background text-foreground"
          : "flex min-h-dvh flex-col bg-background text-foreground"
      }
    >
      <header className="sticky top-0 z-30 shrink-0 w-full border-b border-border bg-background">
        <div className="mx-auto flex h-16 w-full max-w-5xl items-center justify-between gap-3 px-4 sm:px-6">
          <BrandLogo href="/chat" />

          <nav
            aria-label="Primary"
            className="hidden md:flex items-center gap-1"
          >
            {navItems.map((item) => {
              const isActive = pathname === item.href;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  aria-current={isActive ? "page" : undefined}
                  className={`rounded-full px-4 py-2 text-sm transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)] focus-visible:ring-offset-2 ${
                    isActive
                      ? "bg-[var(--brand-tint)] font-semibold text-[var(--brand-primary)]"
                      : "font-medium text-muted-foreground hover:bg-accent hover:text-foreground"
                  }`}
                >
                  {item.label}
                </Link>
              );
            })}
          </nav>

          <div className="flex items-center gap-2">
            <AccountMenu />
          </div>
        </div>
      </header>

      {isChatRoute ? (
        <main
          className={`mx-auto flex w-full ${maxWidthClass} flex-1 min-h-0 flex-col px-4 pt-4 pb-22 sm:px-6 md:py-5`}
        >
          {children}
        </main>
      ) : (
        <main
          className={`mx-auto w-full ${maxWidthClass} flex-1 px-4 pt-6 pb-24 sm:px-6 md:py-8`}
        >
          {children}
        </main>
      )}

      <nav
        aria-label="Primary mobile"
        className="md:hidden fixed inset-x-0 bottom-0 z-30"
      >
        <div className="relative h-14 w-full">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 flex items-stretch"
          >
            <div className="-mr-px h-full flex-1 border-t border-border bg-card" />
            <svg
              width="104"
              height="56"
              viewBox="0 0 104 56"
              fill="none"
              className="relative z-10 h-14 w-[104px] shrink-0"
            >
              <path
                d="M 0 0.5 L 4 0.5 C 20 0.5 26 34 52 34 C 78 34 84 0.5 100 0.5 L 104 0.5 L 104 56 L 0 56 Z"
                className="fill-card"
              />
              <path
                d="M 0 0.5 L 4 0.5 C 20 0.5 26 34 52 34 C 78 34 84 0.5 100 0.5 L 104 0.5"
                className="stroke-border"
                strokeWidth="1"
              />
            </svg>
            <div className="-ml-px h-full flex-1 border-t border-border bg-card" />
          </div>

          <ul className="relative z-20 grid h-full grid-cols-[1fr_104px_1fr] items-center">
            <li className="flex h-full items-center justify-center">
              <Link
                href="/summary"
                aria-current={pathname === "/summary" ? "page" : undefined}
                className={`flex min-h-11 min-w-16 flex-col items-center justify-center gap-0.5 rounded-full px-3 py-1 text-[11px] transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)] focus-visible:ring-offset-2 ${
                  pathname === "/summary"
                    ? "font-semibold text-[var(--brand-primary)]"
                    : "font-medium text-muted-foreground hover:text-foreground"
                }`}
              >
                <BarChart3
                  className="size-4 shrink-0 stroke-[1.75]"
                  aria-hidden="true"
                />
                <span>{t.nav.summary}</span>
              </Link>
            </li>

            <li className="relative flex h-full flex-col items-center justify-end pb-2">
              <Link
                href="/chat"
                aria-label={t.nav.chat}
                title={t.nav.chat}
                aria-current={pathname === "/chat" ? "page" : undefined}
                className="absolute -top-5 left-1/2 flex size-12 -translate-x-1/2 items-center justify-center rounded-full bg-[var(--brand-primary)] text-[var(--brand-primary-text)] transition-colors duration-150 hover:bg-[var(--brand-primary-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-background"
              >
                <NourikoMark
                  className="size-7 shrink-0"
                  aria-hidden="true"
                />
              </Link>
              <span
                aria-hidden="true"
                className={`size-1.5 rounded-full transition-opacity duration-150 ${
                  pathname === "/chat"
                    ? "bg-[var(--brand-primary)] opacity-100"
                    : "bg-transparent opacity-0"
                }`}
              />
            </li>

            <li className="flex h-full items-center justify-center">
              <Link
                href="/history"
                aria-current={pathname === "/history" ? "page" : undefined}
                className={`flex min-h-11 min-w-16 flex-col items-center justify-center gap-0.5 rounded-full px-3 py-1 text-[11px] transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)] focus-visible:ring-offset-2 ${
                  pathname === "/history"
                    ? "font-semibold text-[var(--brand-primary)]"
                    : "font-medium text-muted-foreground hover:text-foreground"
                }`}
              >
                <History
                  className="size-4 shrink-0 stroke-[1.75]"
                  aria-hidden="true"
                />
                <span>{t.nav.history}</span>
              </Link>
            </li>
          </ul>
        </div>
        <div
          aria-hidden="true"
          className="h-[env(safe-area-inset-bottom)] w-full bg-card"
        />
      </nav>
    </div>
  );
}
