"use client";

import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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

  const strengthBarColor =
    strength === "weak"
      ? "bg-[var(--score-low-text)]"
      : strength === "fair"
        ? "bg-[var(--score-fair-text)]"
        : "bg-[var(--brand-primary)]";

  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-sm font-medium text-foreground">
        {label}
      </Label>
      <div className="flex items-center gap-2">
        <Input
          id={id}
          name={name}
          type={visible ? "text" : "password"}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          autoComplete={autoComplete}
          required={required}
          minLength={minLength}
          disabled={disabled}
          className="h-11 rounded-2xl bg-card px-3.5 text-sm shadow-none"
        />
        <Button
          type="button"
          variant="outline"
          onClick={() => setVisible((prev) => !prev)}
          disabled={disabled}
          aria-pressed={visible}
          aria-label={visible ? t.common.hidePassword : t.common.showPassword}
          className="h-11 shrink-0 rounded-full px-3.5 text-xs shadow-none"
        >
          {visible ? (
            <EyeOff className="size-4 stroke-[1.75]" aria-hidden="true" />
          ) : (
            <Eye className="size-4 stroke-[1.75]" aria-hidden="true" />
          )}
          <span className="hidden sm:inline">
            {visible ? t.common.hidePassword : t.common.showPassword}
          </span>
        </Button>
      </div>

      {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}

      {showStrength && strength !== "empty" ? (
        <div className="space-y-1.5 pt-0.5" aria-live="polite">
          <div className="grid grid-cols-3 gap-1.5" aria-hidden="true">
            {[1, 2, 3].map((segment) => (
              <div
                key={segment}
                className={`h-1.5 rounded-full transition-colors duration-150 ${
                  segment <= filledSegments ? strengthBarColor : "bg-muted"
                }`}
              />
            ))}
          </div>
          <p className="text-xs text-muted-foreground">
            {t.common.passwordStrengthLabel}:{" "}
            <span className="font-medium text-foreground">{strengthText}</span>
          </p>
        </div>
      ) : null}
    </div>
  );
}
