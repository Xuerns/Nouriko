"use client";

import Image from "next/image";
import Link from "next/link";
import { useI18n } from "@/components/providers/I18nProvider";

interface BrandLogoProps {
  href: string;
  isCurrentPage?: boolean;
}

export function BrandLogo({ href, isCurrentPage = false }: BrandLogoProps) {
  const { t } = useI18n();

  return (
    <Link
      href={href}
      aria-current={isCurrentPage ? "page" : undefined}
      className="inline-flex items-center gap-2 text-base font-semibold tracking-tight text-neutral-900 dark:text-neutral-100"
    >
      <Image
        src="/logo.png"
        alt=""
        width={24}
        height={24}
        priority
        aria-hidden="true"
        className="h-6 w-6 shrink-0 object-contain"
      />
      <span>{t.common.brand}</span>
    </Link>
  );
}
