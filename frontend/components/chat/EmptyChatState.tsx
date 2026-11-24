"use client";

import { useSyncExternalStore } from "react";
import Image from "next/image";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";

type TimeSlot = "breakfast" | "lunch" | "afternoon" | "dinner" | "lateNight";

function getLocalTimeSlot(): TimeSlot {
  const hour = new Date().getHours();
  if (hour >= 5 && hour < 11) return "breakfast";
  if (hour >= 11 && hour < 15) return "lunch";
  if (hour >= 15 && hour < 18) return "afternoon";
  if (hour >= 18 && hour < 23) return "dinner";
  return "lateNight";
}

function subscribeNoop() {
  return () => {};
}

function getServerTimeSlot(): TimeSlot {
  return "lunch";
}

interface EmptyChatStateProps {
  displayName?: string;
}

export function EmptyChatState({ displayName }: EmptyChatStateProps) {
  const { t } = useI18n();
  const trimmedName = displayName?.trim() ?? "";
  const heading = trimmedName
    ? t.chat.emptyHeadingNamed.replace("{name}", trimmedName)
    : t.chat.emptyHeading;

  return (
    <div className="flex h-full flex-col items-center justify-center px-4 text-center">
      <div className="flex max-w-lg flex-col items-center space-y-3">
        <Image
          src="/logo.png"
          alt={t.common.brand}
          width={56}
          height={56}
          priority
          className="size-14 object-contain"
        />
        <div className="space-y-2">
          <h2 className="text-xl font-semibold tracking-[-0.015em] text-foreground sm:text-2xl">
            {heading}
          </h2>
          <p className="text-sm leading-[1.6] text-muted-foreground sm:text-[15px]">
            {t.chat.emptyDescription}
          </p>
        </div>
      </div>
    </div>
  );
}

interface PromptSuggestionsProps {
  onSelectPrompt: (prompt: string) => void;
  disabled?: boolean;
}

export function PromptSuggestions({
  onSelectPrompt,
  disabled = false,
}: PromptSuggestionsProps) {
  const { t } = useI18n();
  const timeSlot = useSyncExternalStore(
    subscribeNoop,
    getLocalTimeSlot,
    getServerTimeSlot
  );

  const suggestions = [t.chat.timeAwareExamples[timeSlot], ...t.chat.examples];

  return (
    <div
      aria-label={t.chat.examplesHeading}
      className="no-scrollbar flex flex-nowrap items-center gap-2 overflow-x-auto py-0.5"
    >
      {suggestions.map((item) => (
        <Button
          key={item.label}
          type="button"
          variant="outline"
          disabled={disabled}
          onClick={() => onSelectPrompt(item.prompt)}
          className="h-9 shrink-0 whitespace-nowrap rounded-full border-border bg-card px-3.5 text-xs font-medium text-muted-foreground shadow-none hover:bg-accent hover:text-foreground"
        >
          {item.label}
        </Button>
      ))}
    </div>
  );
}
