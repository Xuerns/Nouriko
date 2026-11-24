"use client";

import type { ReactNode } from "react";

interface AlertProps {
  variant: "error" | "success" | "info";
  children: ReactNode;
  className?: string;
}

export function Alert({ variant, children, className = "" }: AlertProps) {
  const variantClass =
    variant === "error"
      ? "bg-[var(--error-bg)] text-[var(--error-text)]"
      : variant === "success"
        ? "bg-[var(--score-good-bg)] text-[var(--score-good-text)]"
        : "bg-[var(--brand-accent-tint)] text-foreground border border-border";

  return (
    <div
      role="alert"
      aria-live="polite"
      className={`rounded-2xl shadow-none px-4 py-3 text-sm leading-relaxed ${variantClass} ${className}`}
    >
      {children}
    </div>
  );
}
