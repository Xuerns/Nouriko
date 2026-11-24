"use client";

import Link from "next/link";
import FadeContent from "@/components/FadeContent";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";

export default function LandingPage() {
  const { isAuthenticated, isLoading, profile } = useAuth();
  const { t } = useI18n();

  const primaryHref = profile?.onboarding_completed ? "/chat" : "/onboarding";
  const primaryLabel = profile?.onboarding_completed
    ? t.landing.ctaOpenChat
    : t.landing.ctaContinueOnboarding;

  return (
    <div className="space-y-12 py-2 sm:py-6">
      <FadeContent trigger="mount" duration={0.4} once>
        <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 sm:p-10">
          <div className="space-y-6">
            <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-[var(--brand-primary)]">
              {t.landing.eyebrow}
            </p>
            <div className="space-y-3">
              <h1 className="text-[28px] font-semibold leading-[1.2] tracking-[-0.02em] text-foreground sm:text-[40px]">
                {t.landing.title}
              </h1>
              <p className="max-w-2xl text-[15px] leading-[1.6] text-muted-foreground sm:text-base">
                {t.landing.tagline}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-3 pt-1">
              {isLoading ? null : isAuthenticated ? (
                <Button
                  render={<Link href={primaryHref} />}
                  size="lg"
                  className="h-11 rounded-full px-6 text-sm font-medium shadow-none"
                >
                  {primaryLabel}
                </Button>
              ) : (
                <>
                  <Button
                    render={<Link href="/register" />}
                    size="lg"
                    className="h-11 rounded-full px-6 text-sm font-medium shadow-none"
                  >
                    {t.landing.ctaRegister}
                  </Button>
                  <Button
                    render={<Link href="/login" />}
                    variant="outline"
                    size="lg"
                    className="h-11 rounded-full px-6 text-sm font-medium shadow-none"
                  >
                    {t.landing.ctaLogin}
                  </Button>
                </>
              )}
            </div>
          </div>
        </Card>
      </FadeContent>

      <FadeContent duration={0.4} delay={0.05} once>
        <section className="space-y-5">
          <h2 className="text-lg font-semibold tracking-[-0.01em] text-foreground sm:text-xl">
            {t.landing.capabilitiesHeading}
          </h2>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            {t.landing.capabilities.map((item) => (
              <Card
                key={item.title}
                className="rounded-3xl border border-border bg-card py-5 shadow-none ring-0"
              >
                <CardHeader className="px-6">
                  <CardTitle className="text-base font-semibold text-foreground">
                    {item.title}
                  </CardTitle>
                </CardHeader>
                <CardContent className="px-6">
                  <p className="text-sm leading-relaxed text-muted-foreground">
                    {item.description}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </section>
      </FadeContent>

      <Separator className="bg-border" />

      <FadeContent duration={0.4} delay={0.1} once>
        <section className="space-y-5">
          <h2 className="text-lg font-semibold tracking-[-0.01em] text-foreground sm:text-xl">
            {t.landing.stepsHeading}
          </h2>
          <ol className="grid grid-cols-1 gap-4 md:grid-cols-3">
            {t.landing.steps.map((step) => (
              <li key={step.step}>
                <Card className="h-full rounded-3xl border border-border bg-card py-5 shadow-none ring-0">
                  <CardHeader className="px-6 space-y-1.5">
                    <p className="text-xs font-semibold tabular-nums text-[var(--brand-accent)]">
                      {step.step}
                    </p>
                    <CardTitle className="text-base font-semibold text-foreground">
                      {step.title}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="px-6">
                    <p className="text-sm leading-relaxed text-muted-foreground">
                      {step.description}
                    </p>
                  </CardContent>
                </Card>
              </li>
            ))}
          </ol>
        </section>
      </FadeContent>
    </div>
  );
}
