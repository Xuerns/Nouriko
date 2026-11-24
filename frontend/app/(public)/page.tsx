"use client";

import { BottomCtaBanner } from "@/components/landing/BottomCtaBanner";
import { CapabilitiesShowcase } from "@/components/landing/CapabilitiesShowcase";
import { LandingHero } from "@/components/landing/LandingHero";
import { WorkflowShowcase } from "@/components/landing/WorkflowShowcase";
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
    <div className="space-y-9 sm:space-y-11">
      <LandingHero
        t={t}
        isAuthenticated={isAuthenticated}
        isLoading={isLoading}
        primaryHref={primaryHref}
        primaryLabel={primaryLabel}
      />

      <CapabilitiesShowcase t={t} />

      <WorkflowShowcase t={t} />

      <BottomCtaBanner
        t={t}
        isAuthenticated={isAuthenticated}
        isLoading={isLoading}
        primaryHref={primaryHref}
        primaryLabel={primaryLabel}
      />
    </div>
  );
}
