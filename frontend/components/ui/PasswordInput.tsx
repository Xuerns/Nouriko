"use client";

import { useState } from "react";
import { Eye, EyeOff, Lock } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { Label } from "@/components/ui/label";
import { evaluatePasswordStrength } from "@/lib/password";
import { cn } from "@/lib/utils";

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
  placeholder?: string;
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
  placeholder,
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

  const segmentColorClass =
    strength === "weak"
      ? "bg-[var(--brand-accent)]"
      : strength === "fair"
        ? "bg-[#C78A3B]"
        : "bg-[var(--brand-primary)]";

  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-sm font-medium text-foreground">
        {label}
      </Label>
      <div className="relative">
        <Lock
          className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground"
          strokeWidth={1.75}
          aria-hidden="true"
        />
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
          placeholder={placeholder}
          className="h-12 w-full rounded-2xl border border-border bg-card pl-10 pr-11 text-sm text-foreground placeholder:text-muted-foreground shadow-none transition-colors outline-none focus-visible:border-[var(--brand-primary)] focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)]/20 disabled:pointer-events-none disabled:opacity-50"
        />
        <Button
          type="button"
          variant="outline"
          onClick={() => setVisible((prev) => !prev)}
          disabled={disabled}
          aria-label={visible ? t.common.hidePassword : t.common.showPassword}
          className="absolute right-1.5 top-1/2 -translate-y-1/2 flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)] disabled:pointer-events-none"
        >
          {visible ? (
            <EyeOff className="h-4 w-4" strokeWidth={1.75} />
          ) : (
            <Eye className="h-4 w-4" strokeWidth={1.75} />
          )}
        </button>
      </div>

      {hint ? (
        <p className="text-xs text-muted-foreground">{hint}</p>
      ) : null}

      {showStrength && strength !== "empty" ? (
        <div className="space-y-1.5 pt-1" aria-live="polite">
          <div className="grid grid-cols-3 gap-2" aria-hidden="true">
            {[1, 2, 3].map((segment) => (
              <div
                key={segment}
                className={cn(
                  "h-1.5 rounded-full transition-colors duration-150",
                  segment <= filledSegments
                    ? segmentColorClass
                    : "bg-muted"
                )}
              />
            ))}
          </div>
          <p className="text-xs text-muted-foreground">
            {t.common.passwordStrengthLabel}:{" "}
            <span className="font-medium text-foreground">
              {strengthText}
            </span>
          </p>
        </div>
      ) : null}
    </div>
  );
}
