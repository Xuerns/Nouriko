"use client";

import { useI18n } from "@/components/providers/I18nProvider";

interface EmptyChatStateProps {
  onSelectPrompt: (prompt: string) => void;
  disabled?: boolean;
}

export function EmptyChatState({
  onSelectPrompt,
  disabled = false,
}: EmptyChatStateProps) {
  const { t } = useI18n();

  return (
    <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-5 sm:p-6 space-y-5">
      <div className="space-y-1.5">
        <h2 className="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
          {t.chat.emptyHeading}
        </h2>
        <p className="text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed">
          {t.chat.emptyDescription}
        </p>
      </div>

      <div className="space-y-2.5">
        <p className="font-mono text-xs uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
          {t.chat.examplesHeading}
        </p>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {t.chat.examples.map((item) => (
            <button
              key={item.label}
              type="button"
              disabled={disabled}
              onClick={() => onSelectPrompt(item.prompt)}
              className="rounded-md border border-neutral-300 dark:border-neutral-700 p-3.5 text-left transition-colors duration-150 hover:border-neutral-500 dark:hover:border-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
            >
              <span className="block font-mono text-xs uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
                {item.label}
              </span>
              <span className="mt-1 block text-sm text-neutral-900 dark:text-neutral-100 leading-snug">
                {item.prompt}
              </span>
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
