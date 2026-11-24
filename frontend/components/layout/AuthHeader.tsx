"use client";

import { BrandLogo } from "@/components/layout/BrandLogo";
import { UtilityControls } from "@/components/layout/UtilityControls";

export function AuthHeader() {
  return (
    <header className="w-full border-b border-border bg-background">
      <div className="mx-auto flex h-16 w-full max-w-4xl items-center justify-between gap-3 px-4 sm:px-6">
        <BrandLogo href="/" />
        <UtilityControls />
      </div>
    </header>
  );
}
