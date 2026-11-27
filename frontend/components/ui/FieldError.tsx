"use client";

import { AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface FieldErrorProps {
  id: string;
  message?: string | null;
  className?: string;
}

export function FieldError({ id, message, className }: FieldErrorProps) {
  if (!message) return null;

  return (
    <p
      id={id}
      className={cn(
        "flex items-start gap-1.5 text-xs leading-snug text-[var(--error-text)]",
        className
      )}
    >
      <AlertCircle
        className="mt-0.5 size-3.5 shrink-0 stroke-[1.75]"
        aria-hidden="true"
      />
      <span>{message}</span>
    </p>
  );
}
