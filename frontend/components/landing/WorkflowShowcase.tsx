"use client";

import { useState } from "react";
import Image from "next/image";
import FadeContent from "@/components/FadeContent";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import type { Dictionary } from "@/lib/i18n/types";

interface WorkflowShowcaseProps {
  t: Dictionary;
}

export function WorkflowShowcase({ t }: WorkflowShowcaseProps) {
  const [activeStep, setActiveStep] = useState(0);
  const previews = t.landing.stepPreviews;

  return (
    <FadeContent duration={0.34} threshold={0.1} once>
      <section
        id="how-it-works"
        aria-labelledby="landing-steps-heading"
        className="scroll-mt-20 grid grid-cols-1 items-start gap-7 border-t border-border pt-8 sm:pt-10 lg:grid-cols-12 lg:gap-9"
      >
        <div className="space-y-4 lg:sticky lg:top-20 lg:col-span-5">
          <div className="space-y-1.5">
            <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-[var(--brand-accent)]">
              {t.landing.stepsEyebrow}
            </p>
            <h2
              id="landing-steps-heading"
              className="text-[clamp(1.25rem,1.05rem+0.95vw,1.65rem)] font-semibold leading-[1.25] tracking-[-0.015em] text-foreground text-balance"
            >
              {t.landing.stepsHeading}
            </h2>
            <p className="text-[13px] sm:text-sm leading-[1.55] text-muted-foreground text-pretty">
              {t.landing.stepsIntro}
            </p>
          </div>

          <div className="space-y-2.5">
            <div className="overflow-clip rounded-3xl border border-border bg-muted">
              <Image
                src="/images/landing/workflow-daily-desk.jpg"
                alt={t.landing.workflowPhotoAlt}
                width={1200}
                height={896}
                loading="lazy"
                sizes="(min-width: 1024px) 400px, 100vw"
                className="block aspect-[16/10] w-full object-cover"
              />
            </div>

            <Card
              aria-live="polite"
              className="rounded-3xl border border-border bg-card py-3.5 shadow-none ring-0 [--card-spacing:--spacing(4)]"
            >
              <CardContent className="flex min-h-[110px] flex-col justify-between gap-2.5">
                {activeStep === 0 ? (
                  <>
                    <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                      {previews.profile.label}
                    </p>
                    <p className="text-xs tabular-nums text-muted-foreground">
                      {previews.profile.bodyMeta}
                    </p>
                    <div className="flex items-baseline justify-between gap-3 rounded-2xl bg-muted px-3.5 py-2.5">
                      <span className="text-[13px] text-muted-foreground">
                        {previews.profile.targetLabel}
                      </span>
                      <span className="text-base font-semibold tabular-nums text-[var(--brand-primary)]">
                        {previews.profile.targetValue}
                      </span>
                    </div>
                    <p className="text-xs leading-[1.5] tabular-nums text-muted-foreground">
                      {previews.profile.macrosSummary}
                    </p>
                  </>
                ) : activeStep === 1 ? (
                  <>
                    <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                      {previews.chat.label}
                    </p>
                    <div className="rounded-2xl rounded-br-md bg-muted px-3.5 py-2.5 text-[13px] font-medium leading-[1.5] text-foreground">
                      {previews.chat.userMessage}
                    </div>
                    <div className="space-y-1 text-xs leading-[1.5] text-muted-foreground">
                      <p>{previews.chat.attachmentNote}</p>
                      <p className="font-medium text-[var(--brand-primary)]">
                        {previews.chat.replyNote}
                      </p>
                    </div>
                  </>
                ) : (
                  <>
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-muted-foreground">
                        {previews.tracking.label}
                      </p>
                      <Badge className="h-6 rounded-full border-0 bg-[var(--score-good-bg)] px-2.5 text-[11px] font-semibold tabular-nums text-[var(--score-good-text)] shadow-none">
                        {previews.tracking.scoreBadge}
                      </Badge>
                    </div>

                    <div className="space-y-2">
                      <div className="flex items-baseline justify-between gap-2 text-[13px]">
                        <span className="text-muted-foreground">
                          {previews.tracking.caloriesLabel}
                        </span>
                        <span className="font-semibold tabular-nums text-foreground">
                          {previews.tracking.caloriesValue}
                        </span>
                      </div>
                      <Progress
                        value={previews.tracking.progressPercent}
                        aria-label={previews.tracking.caloriesLabel}
                        className="gap-0 [&_[data-slot=progress-track]]:h-2 [&_[data-slot=progress-track]]:bg-muted [&_[data-slot=progress-indicator]]:rounded-full [&_[data-slot=progress-indicator]]:bg-[var(--brand-primary)]"
                      />
                    </div>

                    <div className="flex items-center justify-between gap-2 border-t border-border pt-2.5 text-xs">
                      <span className="text-muted-foreground">
                        {previews.tracking.spendLabel}
                      </span>
                      <span className="font-semibold tabular-nums text-foreground">
                        {previews.tracking.spendValue}
                      </span>
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          </div>
        </div>

        <ol className="divide-y divide-border border-y border-border lg:col-span-7">
          {t.landing.steps.map((step, index) => {
            const isActive = activeStep === index;

            return (
              <li key={step.step} className="py-1.5">
                <button
                  type="button"
                  aria-pressed={isActive}
                  onClick={() => setActiveStep(index)}
                  className={`grid min-h-11 w-full cursor-pointer grid-cols-[2.5rem_1fr] sm:grid-cols-[3.25rem_1fr] items-start gap-3.5 sm:gap-5 rounded-2xl px-3.5 py-3.5 sm:px-4 sm:py-4 text-left transition-colors duration-150 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring ${
                    isActive
                      ? "bg-[var(--bg-subtle)]"
                      : "bg-transparent hover:bg-[var(--bg-subtle)]/60"
                  }`}
                >
                  <span
                    className={`text-xl sm:text-2xl font-semibold leading-none tracking-tight tabular-nums transition-colors duration-150 ${
                      isActive
                        ? "text-[var(--brand-accent)]"
                        : "text-[var(--text-muted)]"
                    }`}
                  >
                    {step.step}
                  </span>

                  <span className="block space-y-1.5">
                    <span className="block text-base font-semibold leading-[1.35] text-foreground text-balance">
                      {step.title}
                    </span>
                    <span className="block text-sm leading-[1.55] text-muted-foreground text-pretty">
                      {step.description}
                    </span>
                    <span
                      className={`inline-block pt-0.5 text-xs font-semibold transition-colors duration-150 ${
                        isActive
                          ? "text-[var(--brand-primary)]"
                          : "text-muted-foreground"
                      }`}
                    >
                      {step.highlight}
                    </span>
                  </span>
                </button>
              </li>
            );
          })}
        </ol>
      </section>
    </FadeContent>
  );
}
