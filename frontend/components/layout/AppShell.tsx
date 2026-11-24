"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { BarChart3, History, MessageSquare } from "lucide-react";
import { AccountMenu } from "@/components/layout/AccountMenu";
import { BrandLogo } from "@/components/layout/BrandLogo";
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
          className={`mx-auto flex w-full ${maxWidthClass} flex-1 min-h-0 flex-col px-4 pt-4 pb-20 sm:px-6 md:py-5`}
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
        className="md:hidden fixed inset-x-0 bottom-0 z-30 border-t border-border bg-background pb-[env(safe-area-inset-bottom)]"
      >
        <ul className="grid grid-cols-3 px-2 py-1.5">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = pathname === item.href;
            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  aria-current={isActive ? "page" : undefined}
                  className={`flex min-h-11 flex-col items-center justify-center gap-1 rounded-full py-1.5 text-[11px] transition-colors duration-150 ${
                    isActive
                      ? "bg-[var(--brand-tint)] font-semibold text-[var(--brand-primary)]"
                      : "font-medium text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <Icon
                    className="size-4 shrink-0 stroke-[1.75]"
                    aria-hidden="true"
                  />
                  <span>{item.label}</span>
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    </div>
  );
}
