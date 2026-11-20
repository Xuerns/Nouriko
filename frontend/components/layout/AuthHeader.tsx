"use client";

import { BrandLogo } from "@/components/layout/BrandLogo";
import { UtilityControls } from "@/components/layout/UtilityControls";

export function AuthHeader() {
  return (
    <header className="w-full border-b border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950">
      <div className="mx-auto flex h-14 w-full max-w-4xl items-center justify-between gap-3 px-4">
        <BrandLogo href="/" />
        <UtilityControls />
      </div>
    </header>
  );
}
