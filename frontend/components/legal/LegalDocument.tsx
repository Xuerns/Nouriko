"use client";

import type { LegalSection } from "@/lib/i18n/types";

const SECTION_HEADING_PREFIX = /^\d+\.\s*/;

function parseSectionHeading(heading: string, index: number) {
  const number = String(index + 1).padStart(2, "0");
  const title = heading.replace(SECTION_HEADING_PREFIX, "").trim();
  return { number, title };
}

function splitLeadItem(text: string): { lead: string | null; body: string } {
  const separatorIndex = text.indexOf(":");
  if (separatorIndex > 0 && separatorIndex < 42) {
    return {
      lead: text.slice(0, separatorIndex).trim(),
      body: text.slice(separatorIndex + 1).trim(),
    };
  }
  return { lead: null, body: text };
}

interface LegalDocumentProps {
  title: string;
  updatedLabel: string;
  sections: LegalSection[];
}

export function LegalDocument({
  title,
  updatedLabel,
  sections,
}: LegalDocumentProps) {
  return (
    <article className="mx-auto w-full max-w-2xl">
      <header className="space-y-1.5 border-b border-border pb-5">
        <h1 className="text-[clamp(1.5rem,1.25rem+1.1vw,2rem)] font-semibold leading-[1.2] tracking-[-0.02em] text-foreground text-balance">
          {title}
        </h1>
        <p className="text-[13px] tabular-nums text-muted-foreground">
          {updatedLabel}
        </p>
      </header>

      <div className="divide-y divide-border">
        {sections.map((section, index) => {
          const { number, title: sectionTitle } = parseSectionHeading(
            section.heading,
            index
          );

          return (
            <section
              key={section.heading}
              className="space-y-3 py-6 last:pb-0"
            >
              <div className="flex items-baseline gap-2.5">
                <span className="font-mono text-xs font-semibold tabular-nums text-[var(--brand-primary)]">
                  {number}
                </span>
                <h2 className="text-base sm:text-[17px] font-semibold leading-[1.35] tracking-[-0.01em] text-foreground">
                  {sectionTitle}
                </h2>
              </div>

              {section.paragraphs.map((paragraph) => (
                <p
                  key={paragraph}
                  className="text-[15px] leading-[1.68] text-muted-foreground text-pretty"
                >
                  {paragraph}
                </p>
              ))}

              {section.items && section.items.length > 0 ? (
                <ul className="list-disc space-y-2 pl-5 text-[15px] leading-[1.65] text-muted-foreground marker:text-muted-foreground/60">
                  {section.items.map((item) => {
                    const { lead, body } = splitLeadItem(item);
                    return (
                      <li key={item} className="text-pretty">
                        {lead ? (
                          <>
                            <span className="font-medium text-foreground">
                              {lead}:
                            </span>{" "}
                            {body}
                          </>
                        ) : (
                          body
                        )}
                      </li>
                    );
                  })}
                </ul>
              ) : null}
            </section>
          );
        })}
      </div>
    </article>
  );
}
