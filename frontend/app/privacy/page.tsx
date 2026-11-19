"use client";

import { useI18n } from "@/components/providers/I18nProvider";

export default function PrivacyPage() {
  const { t } = useI18n();

  return (
    <article className="space-y-6">
      <header className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
          {t.privacy.title}
        </h1>
        <p className="text-xs text-neutral-600 dark:text-neutral-400">
          {t.privacy.updatedLabel}
        </p>
      </header>

      <div className="space-y-4">
        {t.privacy.sections.map((section) => (
          <section
            key={section.heading}
            className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-3"
          >
            <h2 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
              {section.heading}
            </h2>
            {section.paragraphs.map((paragraph) => (
              <p
                key={paragraph}
                className="text-sm leading-relaxed text-neutral-700 dark:text-neutral-300"
              >
                {paragraph}
              </p>
            ))}
            {section.items && section.items.length > 0 ? (
              <ul className="list-disc space-y-1.5 pl-5 text-sm leading-relaxed text-neutral-700 dark:text-neutral-300">
                {section.items.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            ) : null}
          </section>
        ))}
      </div>
    </article>
  );
}
