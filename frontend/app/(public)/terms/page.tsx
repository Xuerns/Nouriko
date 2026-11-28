"use client";

import { LegalDocument } from "@/components/legal/LegalDocument";
import { useI18n } from "@/components/providers/I18nProvider";

export default function TermsPage() {
  const { t } = useI18n();

  return (
    <LegalDocument
      title={t.terms.title}
      updatedLabel={t.terms.updatedLabel}
      sections={t.terms.sections}
    />
  );
}
