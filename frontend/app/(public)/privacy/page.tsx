"use client";

import { useI18n } from "@/components/providers/I18nProvider";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function PrivacyPage() {
  const { t } = useI18n();

  return (
    <article className="mx-auto max-w-2xl space-y-6">
      <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0">
        <header className="space-y-2">
          <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground sm:text-[26px]">
            {t.privacy.title}
          </h1>
          <p className="text-xs text-muted-foreground">
            {t.privacy.updatedLabel}
          </p>
        </header>
      </Card>

      <div className="space-y-4">
        {t.privacy.sections.map((section) => (
          <Card
            key={section.heading}
            className="rounded-3xl border border-border bg-card py-6 shadow-none ring-0"
          >
            <CardHeader className="px-6">
              <CardTitle className="text-base font-semibold text-foreground">
                {section.heading}
              </CardTitle>
            </CardHeader>
            <CardContent className="px-6 space-y-3">
              {section.paragraphs.map((paragraph) => (
                <p
                  key={paragraph}
                  className="text-[15px] leading-[1.6] text-muted-foreground"
                >
                  {paragraph}
                </p>
              ))}
              {section.items && section.items.length > 0 ? (
                <ul className="list-disc space-y-1.5 pl-5 text-[15px] leading-[1.6] text-muted-foreground">
                  {section.items.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              ) : null}
            </CardContent>
          </Card>
        ))}
      </div>
    </article>
  );
}
