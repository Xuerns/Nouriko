"use client";

import { useState } from "react";
import { useI18n } from "@/components/providers/I18nProvider";
import { evaluatePasswordStrength } from "@/lib/password";

interface PasswordInputProps {
  id: string;
  name: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  autoComplete: "current-password" | "new-password";
  required?: boolean;
  minLength?: number;
  disabled?: boolean;
  showStrength?: boolean;
  hint?: string;
}

export function PasswordInput({
  id,
  name,
  label,
  value,
  onChange,
  autoComplete,
  required = true,
  minLength,
  disabled = false,
  showStrength = false,
  hint,
}: PasswordInputProps) {
  const { t } = useI18n();
  const [visible, setVisible] = useState(false);

  const strength = showStrength ? evaluatePasswordStrength(value) : "empty";
  const strengthText =
    strength === "weak"
      ? t.common.passwordWeak
      : strength === "fair"
        ? t.common.passwordFair
        : strength === "strong"
          ? t.common.passwordStrong
          : "";

  const filledSegments =
    strength === "weak"
      ? 1
      : strength === "fair"
        ? 2
        : strength === "strong"
          ? 3
          : 0;

  return (
    <div className="space-y-1.5">
      <label
        htmlFor={id}
        className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
      >
        {label}
      </label>
      <div className="flex items-stretch gap-2">
        <input
          id={id}
          name={name}
          type={visible ? "text" : "password"}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          autoComplete={autoComplete}
          required={required}
          minLength={minLength}
          disabled={disabled}
          className="w-full min-w-0 rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
        />
        <button
          type="button"
          onClick={() => setVisible((prev) => !prev)}
          disabled={disabled}
          aria-pressed={visible}
          className="shrink-0 rounded-md border border-neutral-300 dark:border-neutral-700 px-3 py-2 text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
        >
          {visible ? t.common.hidePassword : t.common.showPassword}
        </button>
      </div>

      {hint ? (
        <p className="text-xs text-neutral-600 dark:text-neutral-400">{hint}</p>
      ) : null}

      {showStrength && strength !== "empty" ? (
        <div className="space-y-1 pt-0.5" aria-live="polite">
          <div className="grid grid-cols-3 gap-1.5" aria-hidden="true">
            {[1, 2, 3].map((segment) => (
              <div
                key={segment}
                className={`h-1.5 rounded-sm border border-neutral-300 dark:border-neutral-700 ${
                  segment <= filledSegments
                    ? "bg-neutral-900 dark:bg-neutral-100"
                    : "bg-transparent"
                }`}
              />
            ))}
          </div>
          <p className="text-xs text-neutral-600 dark:text-neutral-400">
            {t.common.passwordStrengthLabel}:{" "}
            <span className="font-medium text-neutral-900 dark:text-neutral-100">
              {strengthText}
            </span>
          </p>
        </div>
      ) : null}
    </div>
  );
}
