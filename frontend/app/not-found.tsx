"use client";

import Link from "next/link";
import { AppFooter } from "@/components/layout/AppFooter";
import { PublicHeader } from "@/components/layout/PublicHeader";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

export default function NotFoundPage() {
  const { t } = useI18n();

  return (
    <div className="flex min-h-dvh flex-col bg-background text-foreground">
      <PublicHeader />
      <main className="mx-auto flex w-full max-w-4xl flex-1 flex-col justify-center px-4 py-8 sm:px-6">
        <Card className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-4">
          <h1 className="text-xl font-semibold text-foreground">
            {t.notFound.title}
          </h1>
          <p className="text-sm leading-relaxed text-muted-foreground">
            {t.notFound.description}
          </p>
          <div>
            <Button
              render={<Link href="/" />}
              className="h-11 rounded-full px-5 text-sm shadow-none"
            >
              {t.notFound.backHome}
            </Button>
          </div>
        </Card>
      </main>
      <AppFooter />
    </div>
  );
}
