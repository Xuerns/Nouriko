"use client";

import Link from "next/link";
import { useI18n } from "@/components/providers/I18nProvider";

interface AppFooterProps {
  maxWidthClass?: string;
  className?: string;
}

export function AppFooter({
  maxWidthClass = "max-w-4xl",
  className = "",
}: AppFooterProps) {
  const { t } = useI18n();

  return (
    <footer
      className={`w-full border-t border-border bg-background ${className}`.trim()}
    >
      <div
        className={`mx-auto flex w-full ${maxWidthClass} flex-wrap items-center justify-between gap-2 px-4 py-5 text-xs text-muted-foreground sm:px-6`}
      >
        <p>{t.footer.copyright}</p>
        <Link
          href="/privacy"
          className="font-medium text-foreground underline underline-offset-4 transition-colors duration-150 hover:text-primary"
        >
          {t.footer.privacyLink}
        </Link>
      </div>
    </footer>
  );
}
