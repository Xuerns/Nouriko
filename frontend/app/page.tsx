"use client";

import Link from "next/link";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";

export default function LandingPage() {
  const { isAuthenticated, isLoading, profile } = useAuth();
  const { t } = useI18n();

  const primaryHref = profile?.onboarding_completed ? "/chat" : "/onboarding";
  const primaryLabel = profile?.onboarding_completed
    ? t.landing.ctaOpenChat
    : t.landing.ctaContinueOnboarding;

  return (
    <div className="space-y-10">
      <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 sm:p-8 space-y-5">
        <p className="font-mono text-xs uppercase tracking-widest text-neutral-500 dark:text-neutral-400">
          {t.landing.eyebrow}
        </p>
        <div className="space-y-3">
          <h1 className="text-3xl font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
            {t.landing.title}
          </h1>
          <p className="max-w-2xl text-sm sm:text-base leading-relaxed text-neutral-700 dark:text-neutral-300">
            {t.landing.tagline}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3 pt-1">
          {isLoading ? null : isAuthenticated ? (
            <Link
              href={primaryHref}
              className="rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200"
            >
              {primaryLabel}
            </Link>
          ) : (
            <>
              <Link
                href="/register"
                className="rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200"
              >
                {t.landing.ctaRegister}
              </Link>
              <Link
                href="/login"
                className="rounded-md border border-neutral-300 dark:border-neutral-700 px-4 py-2 text-sm font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
              >
                {t.landing.ctaLogin}
              </Link>
            </>
          )}
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
          {t.landing.capabilitiesHeading}
        </h2>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          {t.landing.capabilities.map((item) => (
            <div
              key={item.title}
              className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-5 space-y-2"
            >
              <h3 className="text-base font-medium text-neutral-900 dark:text-neutral-100">
                {item.title}
              </h3>
              <p className="text-sm leading-relaxed text-neutral-700 dark:text-neutral-300">
                {item.description}
              </p>
            </div>
          ))}
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
          {t.landing.stepsHeading}
        </h2>
        <ol className="grid grid-cols-1 gap-4 md:grid-cols-3">
          {t.landing.steps.map((step) => (
            <li
              key={step.step}
              className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-5 space-y-2"
            >
              <p className="font-mono text-xs text-neutral-500 dark:text-neutral-400">
                {step.step}
              </p>
              <h3 className="text-base font-medium text-neutral-900 dark:text-neutral-100">
                {step.title}
              </h3>
              <p className="text-sm leading-relaxed text-neutral-700 dark:text-neutral-300">
                {step.description}
              </p>
            </li>
          ))}
        </ol>
      </section>
    </div>
  );
}
