"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { BrandLogo } from "@/components/layout/BrandLogo";
import { UtilityControls } from "@/components/layout/UtilityControls";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";

export function PublicHeader() {
  const pathname = usePathname();
  const { isAuthenticated, isLoading, profile } = useAuth();
  const { t } = useI18n();

  const appHref = profile?.onboarding_completed ? "/chat" : "/onboarding";

  return (
    <header className="sticky top-0 z-30 w-full border-b border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950">
      <div className="mx-auto flex h-14 w-full max-w-5xl items-center justify-between gap-3 px-4">
        <BrandLogo href="/" isCurrentPage={pathname === "/"} />

        <div className="flex items-center gap-2">
          <nav aria-label="Primary" className="flex items-center gap-2">
            {isLoading ? null : isAuthenticated ? (
              <Link
                href={appHref}
                className="rounded-md bg-neutral-900 px-3 py-1.5 text-xs font-medium text-white transition-colors duration-150 hover:bg-neutral-700 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-300"
              >
                {t.nav.chat}
              </Link>
            ) : (
              <>
                <Link
                  href="/login"
                  className="rounded-md border border-neutral-300 dark:border-neutral-700 px-2.5 py-1.5 text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
                >
                  {t.nav.login}
                </Link>
                <Link
                  href="/register"
                  className="rounded-md bg-neutral-900 px-2.5 py-1.5 text-xs font-medium text-white transition-colors duration-150 hover:bg-neutral-700 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-300"
                >
                  {t.nav.register}
                </Link>
              </>
            )}
          </nav>

          <div className="border-l border-neutral-300 dark:border-neutral-700 pl-2">
            <UtilityControls />
          </div>
        </div>
      </div>
    </header>
  );
}
