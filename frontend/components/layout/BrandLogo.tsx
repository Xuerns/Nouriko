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
      className="inline-flex min-h-11 items-center gap-2.5 rounded-full pr-2 text-base font-semibold tracking-tight text-foreground transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
    >
      <Image
        src="/logo.png"
        alt=""
        width={28}
        height={28}
        aria-hidden="true"
        className="size-7 aspect-square shrink-0 object-contain"
      />
      <span>{t.common.brand}</span>
    </Link>
  );
}
