"use client";

import { useEffect, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { Check } from "lucide-react";
import FadeContent from "@/components/FadeContent";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Separator } from "@/components/ui/separator";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";
import type { ActivityLevel, Gender } from "@/lib/types";

interface TargetPreview {
  calories: number;
  protein: number;
  carbs: number;
  fat: number;
  fiber: number;
}

const ACTIVITY_MULTIPLIERS: Record<ActivityLevel, number> = {
  light: 1.375,
  moderate: 1.55,
  heavy: 1.725,
};

function round1(value: number): number {
  return Math.round(value * 10) / 10;
}

function calculatePreviewTargets(
  heightCm: number,
  weightKg: number,
  age: number,
  gender: Gender,
  activityLevel: ActivityLevel
): TargetPreview | null {
  if (
    !Number.isFinite(heightCm) ||
    heightCm < 100 ||
    heightCm > 250 ||
    !Number.isFinite(weightKg) ||
    weightKg < 25 ||
    weightKg > 300 ||
    !Number.isInteger(age) ||
    age < 13 ||
    age > 100
  ) {
    return null;
  }

  const bmr =
    10 * weightKg +
    6.25 * heightCm -
    5 * age +
    (gender === "male" ? 5 : -161);
  const calories = Math.max(
    800,
    Math.round(bmr * ACTIVITY_MULTIPLIERS[activityLevel])
  );

  return {
    calories,
    protein: round1((calories * 0.2) / 4),
    carbs: round1((calories * 0.5) / 4),
    fat: round1((calories * 0.3) / 9),
    fiber: round1((calories / 1000) * 14),
  };
}

function ActivityLevelBars({
  level,
  active,
}: {
  level: 1 | 2 | 3;
  active: boolean;
}) {
  return (
    <span className="inline-flex items-end gap-0.5" aria-hidden="true">
      {[1, 2, 3].map((bar) => {
        const filled = bar <= level;
        const heightClass =
          bar === 1 ? "h-2" : bar === 2 ? "h-3" : "h-4";
        return (
          <span
            key={bar}
            className={`w-1 rounded-full transition-colors duration-150 ${heightClass} ${
              filled
                ? active
                  ? "bg-[var(--brand-primary)]"
                  : "bg-foreground/70"
                : "bg-border"
            }`}
          />
        );
      })}
    </span>
  );
}

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
  const [touched, setTouched] = useState({
    height: false,
    weight: false,
    age: false,
  });
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

  const parsedHeight = heightCm.trim() === "" ? NaN : Number(heightCm);
  const parsedWeight = weightKg.trim() === "" ? NaN : Number(weightKg);
  const parsedAge = age.trim() === "" ? NaN : Number(age);

  const heightValid =
    Number.isFinite(parsedHeight) && parsedHeight >= 100 && parsedHeight <= 250;
  const weightValid =
    Number.isFinite(parsedWeight) && parsedWeight >= 25 && parsedWeight <= 300;
  const ageValid =
    Number.isInteger(parsedAge) && parsedAge >= 13 && parsedAge <= 100;

  const heightInvalid = touched.height && !heightValid;
  const weightInvalid = touched.weight && !weightValid;
  const ageInvalid = touched.age && !ageValid;

  const filledRequiredCount =
    (heightValid ? 1 : 0) + (weightValid ? 1 : 0) + (ageValid ? 1 : 0);

  const preview = calculatePreviewTargets(
    parsedHeight,
    parsedWeight,
    parsedAge,
    gender,
    activityLevel
  );

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!token) return;
    setError(null);
    setTouched({ height: true, weight: true, age: true });

    if (!heightValid || !weightValid || !ageValid) {
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
      <Card className="mx-auto w-full max-w-5xl rounded-3xl border border-border bg-card p-6 text-sm text-muted-foreground shadow-none ring-0">
        {t.common.loading}
      </Card>
    );
  }

  const genderOptions: { value: Gender; label: string }[] = [
    { value: "male", label: t.onboarding.genderMale },
    { value: "female", label: t.onboarding.genderFemale },
  ];

  const activityOptions: {
    value: ActivityLevel;
    label: string;
    description: string;
    bars: 1 | 2 | 3;
  }[] = [
    {
      value: "light",
      label: t.onboarding.activityLight,
      description: t.onboarding.activityLightDesc,
      bars: 1,
    },
    {
      value: "moderate",
      label: t.onboarding.activityModerate,
      description: t.onboarding.activityModerateDesc,
      bars: 2,
    },
    {
      value: "heavy",
      label: t.onboarding.activityHeavy,
      description: t.onboarding.activityHeavyDesc,
      bars: 3,
    },
  ];

  const isSafeAvatarUrl =
    typeof profile?.profile_url === "string" &&
    profile.profile_url.startsWith("https://api.dicebear.com/");

  const trimmedName = displayName.trim();
  const initials = (trimmedName || profile?.email || "N")
    .slice(0, 1)
    .toUpperCase();

  const bodyMetaParts: string[] = [];
  if (heightValid) bodyMetaParts.push(`${parsedHeight} ${t.onboarding.cmUnit}`);
  if (weightValid) bodyMetaParts.push(`${parsedWeight} ${t.onboarding.kgUnit}`);
  if (ageValid) bodyMetaParts.push(`${parsedAge} ${t.onboarding.yearUnit}`);

  const macroCards = [
    {
      key: "protein",
      label: t.profile.protein,
      value: preview ? `${preview.protein} ${t.profile.gramUnit}` : "-",
      ratio: "20%",
      colorVar: "var(--macro-protein)",
    },
    {
      key: "carbs",
      label: t.profile.carbs,
      value: preview ? `${preview.carbs} ${t.profile.gramUnit}` : "-",
      ratio: "50%",
      colorVar: "var(--macro-carbs)",
    },
    {
      key: "fat",
      label: t.profile.fat,
      value: preview ? `${preview.fat} ${t.profile.gramUnit}` : "-",
      ratio: "30%",
      colorVar: "var(--macro-fat)",
    },
    {
      key: "fiber",
      label: t.profile.fiber,
      value: preview ? `${preview.fiber} ${t.profile.gramUnit}` : "-",
      ratio: "14g/1k",
      colorVar: "var(--macro-fiber)",
    },
  ];

  const inputClass =
    "h-11 rounded-2xl border-[var(--border-strong)] bg-background px-3.5 text-sm shadow-none";

  return (
    <FadeContent trigger="mount" duration={0.35} once>
      <div className="grid grid-cols-1 gap-8 lg:grid-cols-12 lg:items-start">
        <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-6 sm:p-8 lg:col-span-7">
          <div className="flex items-center justify-between gap-3">
            <h1 className="text-[24px] font-semibold tracking-[-0.015em] text-foreground sm:text-[28px]">
              {t.onboarding.title}
            </h1>
            <Badge
              className={`h-7 rounded-full border-0 px-3 text-xs font-semibold tabular-nums ${
                filledRequiredCount === 3
                  ? "bg-[var(--brand-tint)] text-[var(--brand-primary)]"
                  : "bg-muted text-muted-foreground"
              }`}
            >
              {filledRequiredCount}/3
            </Badge>
          </div>

          {error ? <Alert variant="error">{error}</Alert> : null}

          <form
            onSubmit={(e) => void handleSubmit(e)}
            noValidate
            className="space-y-5"
          >
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label
                  htmlFor="onboarding-display-name"
                  className="text-sm font-medium text-foreground"
                >
                  <span>{t.onboarding.displayNameLabel}</span>
                  <span className="font-normal text-muted-foreground">
                    ({t.onboarding.displayNameOptional})
                  </span>
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
                  className={inputClass}
                />
              </div>

              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-foreground">
                  {t.onboarding.genderLabel}
                </Label>
                <RadioGroup
                  value={gender}
                  disabled={submitting}
                  onValueChange={(val) => setGender(val as Gender)}
                  className="grid grid-cols-2 gap-2"
                >
                  {genderOptions.map((opt) => {
                    const selected = gender === opt.value;
                    return (
                      <label
                        key={opt.value}
                        htmlFor={`onboarding-gender-${opt.value}`}
                        className={`relative flex h-11 cursor-pointer items-center justify-center gap-1.5 rounded-2xl border px-3 text-sm transition-colors duration-150 focus-within:ring-2 focus-within:ring-[var(--brand-primary)] focus-within:ring-offset-2 ${
                          selected
                            ? "border-[var(--brand-primary)] bg-[var(--brand-tint)] font-semibold text-[var(--brand-primary)]"
                            : "border-[var(--border-strong)] bg-background font-medium text-foreground hover:bg-accent"
                        }`}
                      >
                        <RadioGroupItem
                          id={`onboarding-gender-${opt.value}`}
                          value={opt.value}
                          className="sr-only"
                        />
                        {selected ? (
                          <Check
                            className="size-3.5 shrink-0 stroke-[2.25]"
                            aria-hidden="true"
                          />
                        ) : null}
                        <span>{opt.label}</span>
                      </label>
                    );
                  })}
                </RadioGroup>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div className="space-y-1.5">
                <Label
                  htmlFor="onboarding-height"
                  className="text-sm font-medium text-foreground"
                >
                  {t.onboarding.heightLabel}
                </Label>
                <div className="relative">
                  <Input
                    id="onboarding-height"
                    name="height_cm"
                    type="number"
                    inputMode="decimal"
                    min={100}
                    max={250}
                    step="0.1"
                    required
                    disabled={submitting}
                    value={heightCm}
                    onChange={(e) => setHeightCm(e.target.value)}
                    onBlur={() =>
                      setTouched((prev) => ({ ...prev, height: true }))
                    }
                    placeholder="170"
                    aria-invalid={heightInvalid}
                    aria-describedby={
                      heightInvalid ? "onboarding-height-error" : undefined
                    }
                    className={`${inputClass} pr-10 tabular-nums`}
                  />
                  <span
                    aria-hidden="true"
                    className="pointer-events-none absolute inset-y-0 right-3.5 flex items-center text-xs text-muted-foreground"
                  >
                    {t.onboarding.cmUnit}
                  </span>
                </div>
                {heightInvalid ? (
                  <p
                    id="onboarding-height-error"
                    role="alert"
                    className="text-xs text-[var(--error-text)]"
                  >
                    {t.onboarding.heightRangeError}
                  </p>
                ) : null}
              </div>

              <div className="space-y-1.5">
                <Label
                  htmlFor="onboarding-weight"
                  className="text-sm font-medium text-foreground"
                >
                  {t.onboarding.weightLabel}
                </Label>
                <div className="relative">
                  <Input
                    id="onboarding-weight"
                    name="weight_kg"
                    type="number"
                    inputMode="decimal"
                    min={25}
                    max={300}
                    step="0.1"
                    required
                    disabled={submitting}
                    value={weightKg}
                    onChange={(e) => setWeightKg(e.target.value)}
                    onBlur={() =>
                      setTouched((prev) => ({ ...prev, weight: true }))
                    }
                    placeholder="60"
                    aria-invalid={weightInvalid}
                    aria-describedby={
                      weightInvalid ? "onboarding-weight-error" : undefined
                    }
                    className={`${inputClass} pr-10 tabular-nums`}
                  />
                  <span
                    aria-hidden="true"
                    className="pointer-events-none absolute inset-y-0 right-3.5 flex items-center text-xs text-muted-foreground"
                  >
                    {t.onboarding.kgUnit}
                  </span>
                </div>
                {weightInvalid ? (
                  <p
                    id="onboarding-weight-error"
                    role="alert"
                    className="text-xs text-[var(--error-text)]"
                  >
                    {t.onboarding.weightRangeError}
                  </p>
                ) : null}
              </div>

              <div className="space-y-1.5">
                <Label
                  htmlFor="onboarding-age"
                  className="text-sm font-medium text-foreground"
                >
                  {t.onboarding.ageLabel}
                </Label>
                <div className="relative">
                  <Input
                    id="onboarding-age"
                    name="age"
                    type="number"
                    inputMode="numeric"
                    min={13}
                    max={100}
                    step={1}
                    required
                    disabled={submitting}
                    value={age}
                    onChange={(e) => setAge(e.target.value)}
                    onBlur={() =>
                      setTouched((prev) => ({ ...prev, age: true }))
                    }
                    placeholder="20"
                    aria-invalid={ageInvalid}
                    aria-describedby={
                      ageInvalid ? "onboarding-age-error" : undefined
                    }
                    className={`${inputClass} pr-10 tabular-nums`}
                  />
                  <span
                    aria-hidden="true"
                    className="pointer-events-none absolute inset-y-0 right-3.5 flex items-center text-xs text-muted-foreground"
                  >
                    {t.onboarding.yearUnit}
                  </span>
                </div>
                {ageInvalid ? (
                  <p
                    id="onboarding-age-error"
                    role="alert"
                    className="text-xs text-[var(--error-text)]"
                  >
                    {t.onboarding.ageRangeError}
                  </p>
                ) : null}
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
                className="grid grid-cols-1 gap-2.5 sm:grid-cols-3"
              >
                {activityOptions.map((opt) => {
                  const selected = activityLevel === opt.value;
                  return (
                    <label
                      key={opt.value}
                      htmlFor={`onboarding-activity-${opt.value}`}
                      className={`relative flex min-h-20 cursor-pointer flex-col justify-between gap-2 rounded-2xl border p-3.5 text-left transition-colors duration-150 focus-within:ring-2 focus-within:ring-[var(--brand-primary)] focus-within:ring-offset-2 ${
                        selected
                          ? "border-[var(--brand-primary)] bg-[var(--brand-tint)]"
                          : "border-[var(--border-strong)] bg-background hover:bg-accent"
                      }`}
                    >
                      <RadioGroupItem
                        id={`onboarding-activity-${opt.value}`}
                        value={opt.value}
                        className="sr-only"
                      />
                      <span className="flex items-center justify-between gap-2">
                        <ActivityLevelBars level={opt.bars} active={selected} />
                        {selected ? (
                          <Check
                            className="size-4 shrink-0 stroke-[2.25] text-[var(--brand-primary)]"
                            aria-hidden="true"
                          />
                        ) : null}
                      </span>
                      <span className="space-y-0.5">
                        <span
                          className={`block text-sm ${
                            selected
                              ? "font-semibold text-[var(--brand-primary)]"
                              : "font-medium text-foreground"
                          }`}
                        >
                          {opt.label}
                        </span>
                        <span className="block text-xs leading-snug text-muted-foreground">
                          {opt.description}
                        </span>
                      </span>
                    </label>
                  );
                })}
              </RadioGroup>
            </div>

            <Button
              type="submit"
              disabled={submitting}
              className="h-12 w-full rounded-full text-sm font-semibold shadow-none"
            >
              {submitting ? t.common.saving : t.onboarding.submitButton}
            </Button>
          </form>
        </Card>

        <aside
          aria-live="polite"
          className="lg:sticky lg:top-24 lg:col-span-5"
        >
          <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5">
            <div className="flex items-center gap-3.5">
              <Avatar className="size-12">
                {isSafeAvatarUrl && profile?.profile_url ? (
                  <AvatarImage
                    src={profile.profile_url}
                    alt={t.profile.avatarAlt}
                  />
                ) : null}
                <AvatarFallback className="text-sm font-semibold">
                  {initials}
                </AvatarFallback>
              </Avatar>
              <div className="min-w-0 space-y-0.5">
                <p className="truncate text-sm font-semibold text-foreground">
                  {trimmedName || profile?.email || t.onboarding.title}
                </p>
                <p className="truncate text-xs tabular-nums text-muted-foreground">
                  {bodyMetaParts.length > 0
                    ? bodyMetaParts.join(" · ")
                    : t.onboarding.previewHeading}
                </p>
              </div>
            </div>

            <div className="rounded-2xl bg-[var(--brand-tint)] p-5 space-y-2">
              <p className="text-xs font-medium text-[var(--brand-primary)]">
                {t.onboarding.previewHeading}
              </p>
              {preview ? (
                <div className="flex items-baseline gap-2">
                  <p className="text-4xl font-semibold tracking-[-0.02em] tabular-nums text-[var(--brand-primary)]">
                    {preview.calories}
                  </p>
                  <span className="text-sm font-medium text-[var(--brand-primary)]/80">
                    {t.profile.kcalUnit}
                  </span>
                </div>
              ) : (
                <p className="text-sm leading-relaxed text-[var(--brand-primary)]/85">
                  {t.onboarding.previewEmptyHint}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <div
                aria-hidden="true"
                className="flex h-2.5 w-full overflow-hidden rounded-full bg-muted"
              >
                <div
                  className="h-full transition-opacity duration-200"
                  style={{
                    width: "50%",
                    backgroundColor: "var(--macro-carbs)",
                    opacity: preview ? 1 : 0.35,
                  }}
                />
                <div
                  className="h-full transition-opacity duration-200"
                  style={{
                    width: "20%",
                    backgroundColor: "var(--macro-protein)",
                    opacity: preview ? 1 : 0.35,
                  }}
                />
                <div
                  className="h-full transition-opacity duration-200"
                  style={{
                    width: "30%",
                    backgroundColor: "var(--macro-fat)",
                    opacity: preview ? 1 : 0.35,
                  }}
                />
              </div>
            </div>

            <Separator className="bg-border" />

            <div className="grid grid-cols-2 gap-2.5">
              {macroCards.map((macro) => (
                <div
                  key={macro.key}
                  className="rounded-2xl border border-border bg-background p-3.5 space-y-2"
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-medium text-muted-foreground">
                      {macro.label}
                    </span>
                    <span className="text-[11px] tabular-nums text-muted-foreground">
                      {macro.ratio}
                    </span>
                  </div>
                  <p className="text-base font-semibold tabular-nums text-foreground">
                    {macro.value}
                  </p>
                  <div
                    aria-hidden="true"
                    className="h-1.5 w-10 rounded-full"
                    style={{
                      backgroundColor: macro.colorVar,
                      opacity: preview ? 1 : 0.4,
                    }}
                  />
                </div>
              ))}
            </div>
          </Card>
        </aside>
      </div>
    </FadeContent>
  );
}
