"use client";

import { useI18n } from "@/components/providers/I18nProvider";

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
      <div className="max-w-lg space-y-1.5">
        <h2 className="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
          {heading}
        </h2>
        <p className="text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed">
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
        <button
          key={item.label}
          type="button"
          disabled={disabled}
          onClick={() => onSelectPrompt(item.prompt)}
          className="rounded-md border border-neutral-300 dark:border-neutral-700 px-3 py-2 text-left transition-colors duration-150 hover:border-neutral-500 dark:hover:border-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
        >
          <span className="block font-mono text-[11px] uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
            {item.label}
          </span>
          <span className="mt-0.5 block text-xs text-neutral-900 dark:text-neutral-100 leading-snug">
            {item.prompt}
          </span>
        </button>
      ))}
    </div>
  );
}
