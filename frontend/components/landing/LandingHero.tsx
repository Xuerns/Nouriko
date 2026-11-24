"use client";

import Image from "next/image";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import FadeContent from "@/components/FadeContent";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { Dictionary } from "@/lib/i18n/types";

interface LandingHeroProps {
  t: Dictionary;
  isAuthenticated: boolean;
  isLoading: boolean;
  primaryHref: string;
  primaryLabel: string;
}

const MACRO_ACCENT_BAR: Record<
  Dictionary["landing"]["heroMealPreview"]["macros"][number]["tone"],
  string
> = {
  calories: "bg-[var(--macro-calories)]",
  protein: "bg-[var(--macro-protein)]",
  carbs: "bg-[var(--macro-carbs)]",
  fat: "bg-[var(--macro-fat)]",
  fiber: "bg-[var(--macro-fiber)]",
};

export function LandingHero({
  t,
  isAuthenticated,
  isLoading,
  primaryHref,
  primaryLabel,
}: LandingHeroProps) {
  const preview = t.landing.heroMealPreview;

  return (
    <section
      aria-labelledby="landing-hero-heading"
      className="grid grid-cols-1 items-center gap-7 lg:grid-cols-12 lg:gap-9"
    >
      <div className="space-y-5 lg:col-span-6">
        <div className="space-y-2.5">
          <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-[var(--brand-primary)]">
            {t.landing.eyebrow}
          </p>
          <h1
            id="landing-hero-heading"
            className="text-[clamp(1.75rem,1.3rem+1.85vw,2.5rem)] font-semibold leading-[1.16] tracking-[-0.022em] text-foreground text-balance"
          >
            {t.landing.title}
          </h1>
          <p className="max-w-lg text-[15px] leading-[1.6] text-muted-foreground text-pretty">
            {t.landing.tagline}
          </p>
        </div>

        <div className="space-y-2.5">
          <div className="flex min-h-11 flex-wrap items-center gap-2.5">
            {isLoading ? (
              <>
                <Skeleton aria-hidden="true" className="h-11 w-32 rounded-full" />
                <Skeleton aria-hidden="true" className="h-11 w-28 rounded-full" />
              </>
            ) : isAuthenticated ? (
              <Button
                size="lg"
                nativeButton={false}
                render={<Link href={primaryHref} />}
                className="min-h-11 rounded-full bg-primary px-5 text-[15px] font-medium text-primary-foreground shadow-none transition-colors duration-150 hover:bg-[var(--brand-primary-hover)] active:translate-y-0"
              >
                <span>{primaryLabel}</span>
                <ArrowRight
                  className="size-4"
                  strokeWidth={1.75}
                  aria-hidden="true"
                />
              </Button>
            ) : (
              <>
                <Button
                  size="lg"
                  nativeButton={false}
                  render={<Link href="/register" />}
                  className="min-h-11 rounded-full bg-primary px-5 text-[15px] font-medium text-primary-foreground shadow-none transition-colors duration-150 hover:bg-[var(--brand-primary-hover)] active:translate-y-0"
                >
                  <span>{t.landing.ctaRegister}</span>
                  <ArrowRight
                    className="size-4"
                    strokeWidth={1.75}
                    aria-hidden="true"
                  />
                </Button>
                <Button
                  variant="outline"
                  size="lg"
                  nativeButton={false}
                  render={<Link href="#how-it-works" />}
                  className="min-h-11 rounded-full border-border bg-card px-5 text-[15px] font-medium text-foreground shadow-none transition-colors duration-150 hover:bg-accent active:translate-y-0"
                >
                  {t.footer.howItWorksLink}
                </Button>
              </>
            )}
          </div>

          {!isLoading && !isAuthenticated ? (
            <p className="text-[13px] text-muted-foreground md:hidden">
              {t.register.hasAccount}{" "}
              <Link
                href="/login"
                className="font-medium text-foreground underline underline-offset-4 transition-colors duration-150 hover:text-[var(--brand-primary)]"
              >
                {t.landing.ctaLogin}
              </Link>
            </p>
          ) : null}
        </div>

        <dl className="grid grid-cols-1 gap-3 border-t border-border pt-4 sm:grid-cols-3 sm:gap-4">
          {t.landing.heroStats.map((stat) => (
            <div key={stat.value} className="space-y-0.5">
              <dt className="text-base font-semibold tracking-tight tabular-nums text-foreground">
                {stat.value}
              </dt>
              <dd className="text-xs leading-[1.45] text-muted-foreground text-pretty">
                {stat.label}
              </dd>
            </div>
          ))}
        </dl>
      </div>

      <FadeContent
        trigger="mount"
        duration={0.36}
        once
        className="space-y-2.5 lg:col-span-6"
      >
        <div className="overflow-clip rounded-3xl border border-border bg-muted">
          <Image
            src="/images/landing/hero-meal.jpg"
            alt={t.landing.heroPhotoAlt}
            width={1200}
            height={896}
            priority
            fetchPriority="high"
            sizes="(min-width: 1024px) 494px, 100vw"
            className="block aspect-[3/2] w-full object-cover"
          />
        </div>

        <Card className="rounded-3xl border border-border bg-card py-3.5 sm:py-4 shadow-none ring-0 [--card-spacing:--spacing(4)] sm:[--card-spacing:--spacing(5)]">
          <CardContent className="space-y-2.5">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div className="min-w-0 space-y-0.5">
                <p className="text-[15px] font-semibold leading-[1.35] text-card-foreground">
                  {preview.name}
                </p>
                <p className="text-xs tabular-nums text-muted-foreground">
                  {preview.estimate}
                </p>
              </div>
              <Badge className="h-6.5 shrink-0 rounded-full border-0 bg-[var(--score-good-bg)] px-2.5 text-xs font-semibold tabular-nums text-[var(--score-good-text)] shadow-none">
                {preview.scoreBadge}
              </Badge>
            </div>

            <div className="grid grid-cols-5 gap-1.5 rounded-2xl bg-muted p-2 sm:gap-2 sm:p-2.5">
              {preview.macros.map((macro) => (
                <div
                  key={macro.label}
                  className="flex flex-col items-center gap-0.5 text-center"
                >
                  <span
                    aria-hidden="true"
                    className={`mb-0.5 h-0.5 w-5 rounded-full ${MACRO_ACCENT_BAR[macro.tone]}`}
                  />
                  <span className="text-sm font-semibold leading-tight tabular-nums text-foreground">
                    {macro.value}
                  </span>
                  <span className="text-[11px] leading-tight text-muted-foreground">
                    {macro.label}
                  </span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </FadeContent>
    </section>
  );
}
