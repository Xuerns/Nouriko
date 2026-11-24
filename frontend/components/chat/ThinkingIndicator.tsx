"use client";

import { useEffect, useState } from "react";
import { Ellipsis } from "lucide-react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";

interface ThinkingIndicatorProps {
  steps: string[];
  intervalMs?: number;
}

export function ThinkingIndicator({
  steps,
  intervalMs = 2600,
}: ThinkingIndicatorProps) {
  const prefersReducedMotion = useReducedMotion();
  const [stepIndex, setStepIndex] = useState(0);

  useEffect(() => {
    if (steps.length <= 1) return;

    const timer = window.setInterval(() => {
      setStepIndex((prev) => (prev + 1) % steps.length);
    }, intervalMs);

    return () => {
      window.clearInterval(timer);
    };
  }, [intervalMs, steps.length]);

  const currentStep = steps[stepIndex % Math.max(steps.length, 1)] ?? "";

  return (
    <div
      role="status"
      aria-live="polite"
      aria-atomic="true"
      className="flex min-h-6 items-center gap-2 px-1 py-0.5 text-sm text-muted-foreground"
    >
      <Ellipsis
        className="size-4 shrink-0 stroke-[2.25] text-[var(--brand-accent)]"
        aria-hidden="true"
      />
      <div className="relative min-w-0 flex-1 overflow-hidden">
        {prefersReducedMotion ? (
          <span className="block truncate font-medium text-muted-foreground">
            {currentStep}
          </span>
        ) : (
          <AnimatePresence mode="wait" initial={false}>
            <motion.span
              key={currentStep}
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -4 }}
              transition={{ duration: 0.22, ease: "easeOut" }}
              className="block truncate font-medium text-muted-foreground"
            >
              {currentStep}
            </motion.span>
          </AnimatePresence>
        )}
      </div>
    </div>
  );
}
