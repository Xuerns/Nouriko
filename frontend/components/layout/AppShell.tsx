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

  const navItems = [
    { href: "/chat", label: t.nav.chat, icon: MessageSquare },
    { href: "/summary", label: t.nav.summary, icon: BarChart3 },
    { href: "/history", label: t.nav.history, icon: History },
  ] as const;

  return (
    <div
      className={
        isChatRoute
          ? "flex h-dvh flex-col overflow-hidden bg-white text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100"
          : "flex min-h-dvh flex-col bg-white text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100"
      }
    >
      <header className="sticky top-0 z-30 shrink-0 w-full border-b border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950">
        <div className="mx-auto flex h-14 w-full max-w-4xl items-center justify-between gap-3 px-4">
          <div className="flex items-center gap-6">
            <BrandLogo href="/chat" />

            <nav
              aria-label="Primary"
              className="hidden md:flex items-center gap-1.5"
            >
              {navItems.map((item) => {
                const isActive = pathname === item.href;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    aria-current={isActive ? "page" : undefined}
                    className={`rounded-md border px-3 py-1.5 text-xs transition-colors duration-150 ${
                      isActive
                        ? "border-neutral-900 dark:border-neutral-100 bg-neutral-100 dark:bg-neutral-900 font-semibold text-neutral-900 dark:text-neutral-100"
                        : "border-transparent font-medium text-neutral-600 dark:text-neutral-400 hover:border-neutral-300 dark:hover:border-neutral-700 hover:text-neutral-900 dark:hover:text-neutral-100"
                    }`}
                  >
                    {item.label}
                  </Link>
                );
              })}
            </nav>
          </div>

          <AccountMenu />
        </div>
      </header>

      {isChatRoute ? (
        <main className="mx-auto flex w-full max-w-4xl flex-1 min-h-0 flex-col px-4 pt-4 pb-20 md:py-5">
          {children}
        </main>
      ) : (
        <main className="mx-auto w-full max-w-4xl flex-1 px-4 pt-6 pb-20 md:py-8">
          {children}
        </main>
      )}

      <nav
        aria-label="Primary mobile"
        className="md:hidden fixed inset-x-0 bottom-0 z-30 border-t border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 pb-[env(safe-area-inset-bottom)]"
      >
        <ul className="grid grid-cols-3">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = pathname === item.href;
            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  aria-current={isActive ? "page" : undefined}
                  className={`flex flex-col items-center justify-center gap-1 border-t-2 py-2.5 text-[11px] transition-colors duration-150 ${
                    isActive
                      ? "border-neutral-900 dark:border-neutral-100 font-semibold text-neutral-900 dark:text-neutral-100"
                      : "border-transparent font-medium text-neutral-600 dark:text-neutral-400 hover:text-neutral-900 dark:hover:text-neutral-100"
                  }`}
                >
                  <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
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
