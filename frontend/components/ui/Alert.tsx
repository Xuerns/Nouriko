"use client";

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface AlertProps {
  variant: "error" | "success" | "info";
  children: ReactNode;
  className?: string;
}

export function Alert({ variant, children, className = "" }: AlertProps) {
  const variantStyles =
    variant === "error"
      ? "bg-[var(--error-bg)] text-[var(--error-text)]"
      : variant === "success"
        ? "bg-[var(--success-bg)] text-[var(--success-text)]"
        : "bg-[var(--info-bg)] text-[var(--info-text)]";

  return (
    <div
      role="alert"
      aria-live="polite"
      className={cn(
        "rounded-2xl border-0 px-4 py-3 text-sm leading-relaxed shadow-none transition-colors",
        variantStyles,
        className
      )}
    >
      {children}
    </div>
  );
}
