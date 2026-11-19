"use client";

import type { ReactNode } from "react";

interface AlertProps {
  variant: "error" | "success" | "info";
  children: ReactNode;
  className?: string;
}

export function Alert({ variant, children, className = "" }: AlertProps) {
  const accentClass =
    variant === "error"
      ? "border-l-4 border-l-red-700 dark:border-l-red-500 text-red-800 dark:text-red-300"
      : variant === "success"
        ? "border-l-4 border-l-neutral-900 dark:border-l-neutral-100 text-neutral-900 dark:text-neutral-100"
        : "border-l-4 border-l-neutral-500 dark:border-l-neutral-400 text-neutral-800 dark:text-neutral-200";

  return (
    <div
      role="alert"
      aria-live="polite"
      className={`rounded-md border border-neutral-300 dark:border-neutral-700 px-3.5 py-2.5 text-sm leading-relaxed ${accentClass} ${className}`}
    >
      {children}
    </div>
  );
}
