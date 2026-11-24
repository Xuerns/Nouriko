"use client";

import { useEffect, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";
import type { ActivityLevel, Gender } from "@/lib/types";

export default function OnboardingPage() {
  const router = useRouter();
  const {
    token,
    profile,
    isAuthenticated,
    isLoading,
    setProfile,
    handleUnauthorized,
  } = useAuth();
  const { language, t } = useI18n();

  const [displayName, setDisplayName] = useState("");
  const [heightCm, setHeightCm] = useState("");
  const [weightKg, setWeightKg] = useState("");
  const [age, setAge] = useState("");
  const [gender, setGender] = useState<Gender>("male");
  const [activityLevel, setActivityLevel] = useState<ActivityLevel>("moderate");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated) {
      router.replace("/login");
      return;
    }
    if (profile?.onboarding_completed) {
      router.replace("/chat");
    }
  }, [isAuthenticated, isLoading, profile, router]);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!token) return;
    setError(null);

    const parsedHeight = Number(heightCm);
    const parsedWeight = Number(weightKg);
    const parsedAge = Number(age);

    if (
      !Number.isFinite(parsedHeight) ||
      !Number.isFinite(parsedWeight) ||
      !Number.isInteger(parsedAge)
    ) {
      setError(getFriendlyErrorMessage("INVALID_INPUT", language));
      return;
    }

    setSubmitting(true);
    try {
      const trimmedName = displayName.trim();
      const res = await apiClient.completeOnboarding(token, {
        ...(trimmedName ? { display_name: trimmedName } : {}),
        height_cm: parsedHeight,
        weight_kg: parsedWeight,
        age: parsedAge,
        gender,
        activity_level: activityLevel,
      });
      setProfile(res.profile);
      router.replace("/chat");
    } catch (err) {
      if (handleUnauthorized(err)) return;
      const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
      setError(getFriendlyErrorMessage(code, language));
    } finally {
      setSubmitting(false);
    }
  }

  if (isLoading || !isAuthenticated || profile?.onboarding_completed) {
    return (
      <Card className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 text-sm text-muted-foreground shadow-none ring-0">
        {t.common.loading}
      </Card>
    );
  }

  const genderLabels: Record<Gender, string> = {
    male: t.onboarding.genderMale,
    female: t.onboarding.genderFemale,
  };

  const activityOptions: { value: ActivityLevel; label: string }[] = [
    { value: "light", label: t.onboarding.activityLight },
    { value: "moderate", label: t.onboarding.activityModerate },
    { value: "heavy", label: t.onboarding.activityHeavy },
  ];

  return (
    <Card className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-6 sm:p-7">
      <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground">
        {t.onboarding.title}
      </h1>

      {error ? <Alert variant="error">{error}</Alert> : null}

      <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
        <div className="space-y-1.5">
          <Label
            htmlFor="onboarding-display-name"
            className="text-sm font-medium text-foreground"
          >
            {t.onboarding.displayNameLabel}
          </Label>
          <Input
            id="onboarding-display-name"
            name="display_name"
            type="text"
            maxLength={50}
            autoComplete="nickname"
            disabled={submitting}
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder={t.onboarding.displayNamePlaceholder}
            className="h-11 rounded-2xl bg-card px-3.5 text-sm shadow-none"
          />
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label
              htmlFor="onboarding-height"
              className="text-sm font-medium text-foreground"
            >
              {t.onboarding.heightLabel}
            </Label>
            <Input
              id="onboarding-height"
              name="height_cm"
              type="number"
              min={100}
              max={250}
              step="0.1"
              required
              disabled={submitting}
              value={heightCm}
              onChange={(e) => setHeightCm(e.target.value)}
              className="h-11 rounded-2xl bg-card px-3.5 text-sm tabular-nums shadow-none"
            />
          </div>

          <div className="space-y-1.5">
            <Label
              htmlFor="onboarding-weight"
              className="text-sm font-medium text-foreground"
            >
              {t.onboarding.weightLabel}
            </Label>
            <Input
              id="onboarding-weight"
              name="weight_kg"
              type="number"
              min={25}
              max={300}
              step="0.1"
              required
              disabled={submitting}
              value={weightKg}
              onChange={(e) => setWeightKg(e.target.value)}
              className="h-11 rounded-2xl bg-card px-3.5 text-sm tabular-nums shadow-none"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label
              htmlFor="onboarding-age"
              className="text-sm font-medium text-foreground"
            >
              {t.onboarding.ageLabel}
            </Label>
            <Input
              id="onboarding-age"
              name="age"
              type="number"
              min={13}
              max={100}
              step={1}
              required
              disabled={submitting}
              value={age}
              onChange={(e) => setAge(e.target.value)}
              className="h-11 rounded-2xl bg-card px-3.5 text-sm tabular-nums shadow-none"
            />
          </div>

          <div className="space-y-1.5">
            <Label
              htmlFor="onboarding-gender"
              className="text-sm font-medium text-foreground"
            >
              {t.onboarding.genderLabel}
            </Label>
            <Select
              value={gender}
              disabled={submitting}
              onValueChange={(val) => setGender((val ?? "male") as Gender)}
            >
              <SelectTrigger
                id="onboarding-gender"
                className="h-11 w-full rounded-2xl border-border bg-card px-3.5 text-sm shadow-none"
              >
                <SelectValue>{genderLabels[gender]}</SelectValue>
              </SelectTrigger>
              <SelectContent className="rounded-2xl border border-border bg-card shadow-none ring-0">
                <SelectItem value="male" className="rounded-xl">
                  {t.onboarding.genderMale}
                </SelectItem>
                <SelectItem value="female" className="rounded-xl">
                  {t.onboarding.genderFemale}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="space-y-2">
          <Label className="text-sm font-medium text-foreground">
            {t.onboarding.activityLabel}
          </Label>
          <RadioGroup
            value={activityLevel}
            disabled={submitting}
            onValueChange={(val) => setActivityLevel(val as ActivityLevel)}
            className="grid grid-cols-3 gap-2"
          >
            {activityOptions.map((opt) => {
              const selected = activityLevel === opt.value;
              return (
                <Label
                  key={opt.value}
                  htmlFor={`onboarding-activity-${opt.value}`}
                  className={`flex min-h-11 cursor-pointer items-center justify-center gap-2 rounded-2xl border px-3 py-2.5 text-xs font-medium transition-colors duration-150 ${
                    selected
                      ? "border-[var(--brand-primary)] bg-[var(--brand-tint)] text-[var(--brand-primary)]"
                      : "border-border bg-card text-foreground hover:bg-accent"
                  }`}
                >
                  <RadioGroupItem
                    id={`onboarding-activity-${opt.value}`}
                    value={opt.value}
                    className="sr-only"
                  />
                  <span>{opt.label}</span>
                </Label>
              );
            })}
          </RadioGroup>
        </div>

        <Button
          type="submit"
          disabled={submitting}
          className="h-11 w-full rounded-full text-sm font-medium shadow-none"
        >
          {submitting ? t.common.saving : t.onboarding.submitButton}
        </Button>
      </form>
    </Card>
  );
}
