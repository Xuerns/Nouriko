"use client";

import Image from "next/image";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import FadeContent from "@/components/FadeContent";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { Dictionary } from "@/lib/i18n/types";

interface BottomCtaBannerProps {
  t: Dictionary;
  isAuthenticated: boolean;
  isLoading: boolean;
  primaryHref: string;
  primaryLabel: string;
}

export function BottomCtaBanner({
  t,
  isAuthenticated,
  isLoading,
  primaryHref,
  primaryLabel,
}: BottomCtaBannerProps) {
  return (
    <FadeContent duration={0.32} threshold={0.05} once>
      <section aria-labelledby="landing-cta-heading">
        <Card className="gap-0 overflow-clip rounded-3xl border border-border bg-[var(--brand-tint)] py-0 shadow-none ring-0">
          <div className="grid grid-cols-1 lg:grid-cols-12">
            <div className="flex flex-col justify-center gap-4 p-6 sm:p-8 lg:col-span-7 lg:p-9">
              <div className="space-y-2">
                <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-[var(--brand-primary)]">
                  {t.landing.bottomCtaEyebrow}
                </p>
                <h2
                  id="landing-cta-heading"
                  className="text-[clamp(1.25rem,1.05rem+0.95vw,1.65rem)] font-semibold leading-[1.25] tracking-[-0.015em] text-foreground text-balance"
                >
                  {t.landing.bottomCtaHeading}
                </h2>
                <p className="max-w-lg text-sm sm:text-[15px] leading-[1.55] text-muted-foreground text-pretty">
                  {isAuthenticated
                    ? t.landing.bottomCtaAuthedDescription
                    : t.landing.bottomCtaDescription}
                </p>
              </div>

              <div className="flex min-h-11 flex-wrap items-center gap-2.5">
                {isLoading ? (
                  <>
                    <Skeleton
                      aria-hidden="true"
                      className="h-11 w-32 rounded-full"
                    />
                    <Skeleton
                      aria-hidden="true"
                      className="h-11 w-28 rounded-full"
                    />
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
                      render={<Link href="/login" />}
                      className="min-h-11 rounded-full border-[var(--border-strong)] bg-card px-5 text-[15px] font-medium text-foreground shadow-none transition-colors duration-150 hover:bg-accent dark:border-[var(--border-strong)] dark:bg-card dark:hover:bg-accent active:translate-y-0"
                    >
                      {t.landing.ctaLogin}
                    </Button>
                  </>
                )}
              </div>
            </div>

            <div className="relative min-h-52 border-t border-border bg-muted sm:min-h-60 lg:col-span-5 lg:min-h-full lg:border-t-0 lg:border-l">
              <Image
                src="/images/landing/cta-fresh-table.jpg"
                alt={t.landing.bottomCtaPhotoAlt}
                fill
                loading="lazy"
                sizes="(min-width: 1024px) 420px, 100vw"
                className="block object-cover"
              />
            </div>
          </div>
        </Card>
      </section>
    </FadeContent>
  );
}
