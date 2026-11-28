"use client";

import Link from "next/link";
import { BrandLogo } from "@/components/layout/BrandLogo";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Separator } from "@/components/ui/separator";

interface AppFooterProps {
  maxWidthClass?: string;
  className?: string;
}

const footerLinkClass =
  "inline-flex min-h-7 sm:min-h-6 items-center text-[13px] sm:text-sm text-muted-foreground transition-colors duration-150 hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring";

export function AppFooter({
  maxWidthClass = "max-w-5xl",
  className = "",
}: AppFooterProps) {
  const { isAuthenticated, isLoading, profile } = useAuth();
  const { t } = useI18n();

  const primaryAppHref = profile?.onboarding_completed ? "/chat" : "/onboarding";
  const primaryAppLabel = profile?.onboarding_completed
    ? t.nav.chat
    : t.landing.ctaContinueOnboarding;

  return (
    <footer
      className={`w-full border-t border-border bg-[var(--bg-subtle)] shadow-none ${className}`.trim()}
    >
      <div
        className={`mx-auto w-full ${maxWidthClass} space-y-4 sm:space-y-6 px-4 py-5 sm:px-6 sm:py-8`}
      >
        <div className="grid grid-cols-1 gap-5 md:grid-cols-[1.4fr_1fr_1fr_1fr] md:gap-6">
          <div className="space-y-1.5 sm:space-y-2">
            <BrandLogo href="/" />
            <p className="max-w-xs text-[13px] sm:text-sm leading-[1.5] text-muted-foreground text-pretty">
              {t.footer.description}
            </p>
            <p className="text-xs leading-[1.45] text-[var(--text-muted)] text-pretty">
              {t.footer.disclaimer}
            </p>
          </div>

          <nav
            aria-label={t.footer.navAriaLabel}
            className="grid grid-cols-2 gap-x-5 gap-y-4 sm:grid-cols-3 sm:gap-6 md:col-span-3 md:grid-cols-3"
          >
            <div className="space-y-1.5 sm:space-y-2">
              <h2 className="text-[11px] font-semibold uppercase tracking-[0.06em] text-foreground">
                {t.footer.exploreHeading}
              </h2>
              <ul className="space-y-0.5">
                <li>
                  <Link href="/" className={footerLinkClass}>
                    {t.footer.homeLink}
                  </Link>
                </li>
                <li>
                  <Link href="/#capabilities" className={footerLinkClass}>
                    {t.footer.capabilitiesLink}
                  </Link>
                </li>
                <li>
                  <Link href="/#how-it-works" className={footerLinkClass}>
                    {t.footer.howItWorksLink}
                  </Link>
                </li>
              </ul>
            </div>

            <div className="space-y-1.5 sm:space-y-2">
              <h2 className="text-[11px] font-semibold uppercase tracking-[0.06em] text-foreground">
                {t.footer.accountHeading}
              </h2>
              <ul className="space-y-0.5">
                {!isLoading && isAuthenticated ? (
                  <>
                    <li>
                      <Link href={primaryAppHref} className={footerLinkClass}>
                        {primaryAppLabel}
                      </Link>
                    </li>
                    {profile?.onboarding_completed ? (
                      <>
                        <li>
                          <Link href="/summary" className={footerLinkClass}>
                            {t.nav.summary}
                          </Link>
                        </li>
                        <li>
                          <Link href="/history" className={footerLinkClass}>
                            {t.nav.history}
                          </Link>
                        </li>
                        <li>
                          <Link href="/profile" className={footerLinkClass}>
                            {t.nav.profile}
                          </Link>
                        </li>
                      </>
                    ) : null}
                  </>
                ) : (
                  <>
                    <li>
                      <Link href="/register" className={footerLinkClass}>
                        {t.nav.register}
                      </Link>
                    </li>
                    <li>
                      <Link href="/login" className={footerLinkClass}>
                        {t.nav.login}
                      </Link>
                    </li>
                    <li>
                      <Link href="/forgot-password" className={footerLinkClass}>
                        {t.login.forgotPasswordLink}
                      </Link>
                    </li>
                  </>
                )}
              </ul>
            </div>

            <div className="space-y-1.5 sm:space-y-2">
              <h2 className="text-[11px] font-semibold uppercase tracking-[0.06em] text-foreground">
                {t.footer.infoHeading}
              </h2>
              <ul className="space-y-0.5">
                <li>
                  <Link href="/privacy" className={footerLinkClass}>
                    {t.footer.privacyLink}
                  </Link>
                </li>
                <li>
                  <Link href="/terms" className={footerLinkClass}>
                    {t.footer.termsLink}
                  </Link>
                </li>
              </ul>
            </div>
          </nav>
        </div>

        <div className="space-y-3 sm:space-y-4">
          <Separator className="bg-border" />
          <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-1.5 text-xs text-muted-foreground">
            <p>{t.footer.copyright}</p>
          </div>
        </div>
      </div>
    </footer>
  );
}
