"use client";

import Image from "next/image";
import FadeContent from "@/components/FadeContent";
import { Badge } from "@/components/ui/badge";
import type { Dictionary } from "@/lib/i18n/types";

interface CapabilitiesShowcaseProps {
  t: Dictionary;
}

const CAPABILITY_IMAGES = [
  {
    src: "/images/landing/capability-ingredients.jpg",
    width: 1264,
    height: 848,
  },
  {
    src: "/images/landing/capability-budget.jpg",
    width: 1264,
    height: 848,
  },
  {
    src: "/images/landing/capability-photo-scan.jpg",
    width: 1264,
    height: 848,
  },
] as const;

export function CapabilitiesShowcase({ t }: CapabilitiesShowcaseProps) {
  return (
    <section
      id="capabilities"
      aria-labelledby="landing-capabilities-heading"
      className="scroll-mt-20 space-y-6 border-t border-border pt-8 sm:space-y-8 sm:pt-10"
    >
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div className="max-w-xl space-y-1.5">
          <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-[var(--brand-primary)]">
            {t.landing.capabilitiesEyebrow}
          </p>
          <h2
            id="landing-capabilities-heading"
            className="text-[clamp(1.25rem,1.05rem+0.95vw,1.65rem)] font-semibold leading-[1.25] tracking-[-0.015em] text-foreground text-balance"
          >
            {t.landing.capabilitiesHeading}
          </h2>
        </div>
      </div>

      <div className="divide-y divide-border">
        {t.landing.capabilities.map((item, index) => {
          const media = CAPABILITY_IMAGES[index] ?? CAPABILITY_IMAGES[0];
          const isReversed = index % 2 === 1;

          return (
            <div
              key={item.number}
              className="py-6 first:pt-0 last:pb-0 sm:py-7"
            >
              <FadeContent duration={0.34} threshold={0.1} once>
                <article className="grid grid-cols-1 items-center gap-5 lg:grid-cols-12 lg:gap-8">
                  <figure
                    className={`overflow-clip rounded-3xl border border-border bg-muted lg:col-span-6 ${
                      isReversed ? "lg:order-2" : "lg:order-1"
                    }`}
                  >
                    <Image
                      src={media.src}
                      alt={item.imageAlt}
                      width={media.width}
                      height={media.height}
                      loading="lazy"
                      sizes="(min-width: 1024px) 496px, 100vw"
                      className="block aspect-[16/10] w-full object-cover"
                    />
                  </figure>

                  <div
                    className={`space-y-3.5 lg:col-span-6 ${
                      isReversed ? "lg:order-1" : "lg:order-2"
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs font-semibold tabular-nums text-[var(--brand-primary)]">
                        {item.number}
                      </span>
                      <span className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                        {item.tag}
                      </span>
                    </div>

                    <div className="space-y-1.5">
                      <h3 className="text-lg sm:text-xl font-semibold leading-[1.3] tracking-[-0.01em] text-foreground text-balance">
                        {item.title}
                      </h3>
                      <p className="text-sm sm:text-[15px] leading-[1.6] text-muted-foreground text-pretty">
                        {item.description}
                      </p>
                    </div>

                    {item.calloutText ? (
                      <div className="space-y-2.5 rounded-2xl border border-border bg-[var(--bg-subtle)] p-3.5">
                        <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                          {item.calloutLabel}
                        </p>
                        <p className="text-[13px] sm:text-sm font-medium leading-[1.5] text-foreground text-pretty">
                          {item.calloutText}
                        </p>
                        {item.chips ? (
                          <div className="flex flex-wrap gap-1.5">
                            {item.chips.map((chip) => (
                              <span
                                key={chip}
                                className="inline-flex items-center rounded-full border border-border bg-card px-2.5 py-0.5 text-xs font-medium text-muted-foreground"
                              >
                                {chip}
                              </span>
                            ))}
                          </div>
                        ) : null}
                      </div>
                    ) : null}

                    {item.budgetRows ? (
                      <div className="space-y-2 rounded-2xl border border-border bg-[var(--brand-accent-tint)] p-3.5">
                        <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-[var(--brand-accent)]">
                          {item.calloutLabel}
                        </p>
                        <dl className="divide-y divide-border/80">
                          {item.budgetRows.map((row) => (
                            <div
                              key={row.label}
                              className="flex items-center justify-between gap-3 py-1.5 first:pt-0 last:pb-0"
                            >
                              <dt className="text-[13px] text-muted-foreground">
                                {row.label}
                              </dt>
                              <dd className="text-[13px] font-semibold tabular-nums text-foreground">
                                {row.value}
                              </dd>
                            </div>
                          ))}
                        </dl>
                      </div>
                    ) : null}

                    {index === 2 ? (
                      <div className="space-y-2 rounded-2xl border border-border bg-card p-3.5">
                        <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                          {item.calloutLabel}
                        </p>
                        <div className="flex flex-wrap items-center gap-1.5">
                          <Badge className="h-6.5 rounded-full border-0 bg-[var(--score-good-bg)] px-2.5 text-xs font-semibold tabular-nums text-[var(--score-good-text)] shadow-none">
                            {t.landing.rubricBadges.good}
                          </Badge>
                          <Badge className="h-6.5 rounded-full border-0 bg-[var(--score-fair-bg)] px-2.5 text-xs font-semibold tabular-nums text-[var(--score-fair-text)] shadow-none">
                            {t.landing.rubricBadges.fair}
                          </Badge>
                          <Badge className="h-6.5 rounded-full border-0 bg-[var(--score-low-bg)] px-2.5 text-xs font-semibold tabular-nums text-[var(--score-low-text)] shadow-none">
                            {t.landing.rubricBadges.low}
                          </Badge>
                        </div>
                      </div>
                    ) : null}
                  </div>
                </article>
              </FadeContent>
            </div>
          );
        })}
      </div>
    </section>
  );
}
