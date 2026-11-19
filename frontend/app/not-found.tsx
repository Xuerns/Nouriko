"use client";

import Link from "next/link";
import { useI18n } from "@/components/providers/I18nProvider";

export default function NotFoundPage() {
  const { t } = useI18n();

  return (
    <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-4">
      <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
        {t.notFound.title}
      </h1>
      <p className="text-sm leading-relaxed text-neutral-700 dark:text-neutral-300">
        {t.notFound.description}
      </p>
      <Link
        href="/"
        className="inline-block rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200"
      >
        {t.notFound.backHome}
      </Link>
    </div>
  );
}
