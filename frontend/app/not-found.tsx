"use client";

import { useSyncExternalStore } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft, Home, MessageSquare } from "lucide-react";
import { AppFooter } from "@/components/layout/AppFooter";
import { PublicHeader } from "@/components/layout/PublicHeader";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";

const emptySubscribe = () => () => {};

function getCanGoBackSnapshot(): boolean {
  if (typeof window === "undefined") return false;
  if (window.history.length <= 1) return false;
  if (!document.referrer) return true;
  try {
    return new URL(document.referrer).origin === window.location.origin;
  } catch {
    return false;
  }
}

export default function NotFoundPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading, profile } = useAuth();
  const { t } = useI18n();

  const canGoBack = useSyncExternalStore(
    emptySubscribe,
    getCanGoBackSnapshot,
    () => false
  );

  const isOnboarded = !isLoading && isAuthenticated && profile?.onboarding_completed;
  const needsOnboarding =
    !isLoading && isAuthenticated && profile !== null && !profile.onboarding_completed;

  const destinationHref = isOnboarded
    ? "/chat"
    : needsOnboarding
      ? "/onboarding"
      : "/";
  const destinationLabel = isOnboarded
    ? t.notFound.backToChat
    : needsOnboarding
      ? t.landing.ctaContinueOnboarding
      : t.notFound.backHome;
  const DestinationIcon = isOnboarded ? MessageSquare : Home;

  return (
    <div className="flex min-h-dvh flex-col bg-background text-foreground">
      <PublicHeader />
      <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col items-center justify-center px-4 py-10 text-center sm:px-6 sm:py-14">
        <div className="flex w-full max-w-md flex-col items-center">
          <Image
            src="/404.png"
            alt=""
            width={1536}
            height={1024}
            priority
            sizes="(max-width: 640px) 260px, 320px"
            className="h-auto w-[260px] select-none sm:w-[320px]"
          />

          <div className="mt-5 space-y-2">
            <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground sm:text-[26px]">
              {t.notFound.title}
            </h1>
            <p className="mx-auto max-w-sm text-sm leading-relaxed text-muted-foreground">
              {t.notFound.description}
            </p>
          </div>

          <div className="mt-6 flex flex-wrap items-center justify-center gap-2.5">
            {canGoBack ? (
              <>
                <Button
                  type="button"
                  onClick={() => router.back()}
                  className="h-11 gap-2 rounded-full bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-none transition-colors duration-150 hover:bg-[var(--brand-primary-hover)]"
                >
                  <ArrowLeft
                    className="size-4 shrink-0 stroke-[1.75]"
                    aria-hidden="true"
                  />
                  <span>{t.notFound.goBack}</span>
                </Button>
                <Button
                  variant="outline"
                  nativeButton={false}
                  render={<Link href={destinationHref} />}
                  className="h-11 gap-2 rounded-full border-border bg-card px-5 text-sm font-medium text-foreground shadow-none transition-colors duration-150 hover:bg-accent"
                >
                  <DestinationIcon
                    className="size-4 shrink-0 stroke-[1.75] text-muted-foreground"
                    aria-hidden="true"
                  />
                  <span>{destinationLabel}</span>
                </Button>
              </>
            ) : (
              <Button
                nativeButton={false}
                render={<Link href={destinationHref} />}
                className="h-11 gap-2 rounded-full bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-none transition-colors duration-150 hover:bg-[var(--brand-primary-hover)]"
              >
                <ArrowLeft
                  className="size-4 shrink-0 stroke-[1.75]"
                  aria-hidden="true"
                />
                <span>{destinationLabel}</span>
              </Button>
            )}
          </div>
        </div>
      </main>
      <AppFooter />
    </div>
  );
}
