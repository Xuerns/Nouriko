"use client";

import { LegalDocument } from "@/components/legal/LegalDocument";
import { useI18n } from "@/components/providers/I18nProvider";

export default function PrivacyPage() {
  const { t } = useI18n();

  return (
    <LegalDocument
      title={t.privacy.title}
      updatedLabel={t.privacy.updatedLabel}
      sections={t.privacy.sections}
    />
  );
}
