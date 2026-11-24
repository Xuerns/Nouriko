"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { BrandLogo } from "@/components/layout/BrandLogo";
import { UtilityControls } from "@/components/layout/UtilityControls";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";

export function PublicHeader() {
  const pathname = usePathname();
  const { isAuthenticated, isLoading, profile } = useAuth();
  const { t } = useI18n();

  const appHref = profile?.onboarding_completed ? "/chat" : "/onboarding";

  return (
    <header className="sticky top-0 z-30 w-full border-b border-border bg-background">
      <div className="mx-auto flex h-16 w-full max-w-5xl items-center justify-between gap-3 px-4 sm:px-6">
        <BrandLogo href="/" isCurrentPage={pathname === "/"} />

        <div className="flex items-center gap-2">
          <nav aria-label="Primary" className="flex items-center gap-2">
            {isLoading ? null : isAuthenticated ? (
              <Button
                render={<Link href={appHref} />}
                size="sm"
                className="h-9 rounded-full px-4 text-xs font-medium shadow-none"
              >
                {t.nav.chat}
              </Button>
            ) : (
              <>
                <Button
                  render={<Link href="/login" />}
                  variant="outline"
                  size="sm"
                  className="h-9 rounded-full px-4 text-xs font-medium shadow-none"
                >
                  {t.nav.login}
                </Button>
                <Button
                  render={<Link href="/register" />}
                  size="sm"
                  className="h-9 rounded-full px-4 text-xs font-medium shadow-none"
                >
                  {t.nav.register}
                </Button>
              </>
            )}
          </nav>

          <div className="border-l border-border pl-2">
            <UtilityControls />
          </div>
        </div>
      </div>
    </header>
  );
}
