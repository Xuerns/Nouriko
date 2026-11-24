"use client";

import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";

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
      <div className="max-w-lg space-y-2">
        <h2 className="text-xl font-semibold tracking-[-0.01em] text-foreground">
          {heading}
        </h2>
        <p className="text-[15px] leading-[1.6] text-muted-foreground">
          {t.chat.emptyDescription}
        </p>
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
  const quickExamples = t.chat.examples.slice(0, 2);

  return (
    <div
      aria-label={t.chat.examplesHeading}
      className="grid grid-cols-1 gap-2 sm:grid-cols-2"
    >
      {quickExamples.map((item) => (
        <Button
          key={item.label}
          type="button"
          variant="outline"
          disabled={disabled}
          onClick={() => onSelectPrompt(item.prompt)}
          className="h-auto min-h-11 flex-col items-start justify-start rounded-2xl border-border bg-card px-4 py-3 text-left whitespace-normal shadow-none hover:bg-accent"
        >
          <span className="block text-[11px] font-semibold uppercase tracking-[0.06em] text-[var(--brand-primary)]">
            {item.label}
          </span>
          <span className="mt-1 block text-xs font-normal leading-snug text-foreground">
            {item.prompt}
          </span>
        </Button>
      ))}
    </div>
  );
}
