"use client";

import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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

  const toggleLabel = visible ? t.common.hidePassword : t.common.showPassword;

  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-xs font-medium text-foreground">
        {label}
      </Label>
      <div className="relative">
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
          placeholder={placeholder}
          className="h-10 rounded-2xl bg-card pr-10 pl-3.5 text-sm shadow-none [&::-ms-clear]:hidden [&::-ms-reveal]:hidden"
        />
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          onClick={() => setVisible((prev) => !prev)}
          disabled={disabled}
          aria-pressed={visible}
          aria-label={toggleLabel}
          title={toggleLabel}
          className="absolute top-1/2 right-1 size-8 -translate-y-1/2 rounded-full text-muted-foreground shadow-none hover:bg-accent hover:text-foreground"
        >
          {visible ? (
            <EyeOff className="size-4 stroke-[1.75]" aria-hidden="true" />
          ) : (
            <Eye className="size-4 stroke-[1.75]" aria-hidden="true" />
          )}
        </Button>
      </div>

      {hint ? (
        <p className="text-[11px] leading-snug text-muted-foreground">{hint}</p>
      ) : null}

      {showStrength && strength !== "empty" ? (
        <div className="space-y-1 pt-0.5" aria-live="polite">
          <div className="grid grid-cols-3 gap-1.5" aria-hidden="true">
            {[1, 2, 3].map((segment) => (
              <div
                key={segment}
                className={cn(
                  "h-1.5 rounded-full transition-colors duration-150",
                  segment <= filledSegments ? segmentColorClass : "bg-muted"
                )}
              />
            ))}
          </div>
          <p className="text-[11px] text-muted-foreground">
            {t.common.passwordStrengthLabel}:{" "}
            <span className="font-medium text-foreground">{strengthText}</span>
          </p>
        </div>
      ) : null}
    </div>
  );
}
