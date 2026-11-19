"use client";

import Link from "next/link";
import { useI18n } from "@/components/providers/I18nProvider";

export function AppFooter() {
  const { t } = useI18n();

  return (
    <footer className="w-full border-t border-neutral-300 dark:border-neutral-700">
      <div className="mx-auto flex w-full max-w-4xl flex-wrap items-center justify-between gap-2 px-4 py-4 text-xs text-neutral-600 dark:text-neutral-400">
        <p>{t.footer.copyright}</p>
        <Link
          href="/privacy"
          className="underline underline-offset-4 transition-colors duration-150 hover:text-neutral-900 dark:hover:text-neutral-100"
        >
          {t.footer.privacyLink}
        </Link>
      </div>
    </footer>
  );
}
