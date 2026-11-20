"use client";

import { useEffect, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
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
      <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
        {t.common.loading}
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-6">
      <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
        {t.onboarding.title}
      </h1>

      {error ? <Alert variant="error">{error}</Alert> : null}

      <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
        <div className="space-y-1.5">
          <label
            htmlFor="onboarding-display-name"
            className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
          >
            {t.onboarding.displayNameLabel}
          </label>
          <input
            id="onboarding-display-name"
            name="display_name"
            type="text"
            maxLength={50}
            autoComplete="nickname"
            disabled={submitting}
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder={t.onboarding.displayNamePlaceholder}
            className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
          />
        </div>

        <div className="space-y-1.5">
          <label
            htmlFor="onboarding-height"
            className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
          >
            {t.onboarding.heightLabel}
          </label>
          <input
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
            className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
          />
        </div>

        <div className="space-y-1.5">
          <label
            htmlFor="onboarding-weight"
            className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
          >
            {t.onboarding.weightLabel}
          </label>
          <input
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
            className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
          />
        </div>

        <div className="space-y-1.5">
          <label
            htmlFor="onboarding-age"
            className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
          >
            {t.onboarding.ageLabel}
          </label>
          <input
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
            className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
          />
        </div>

        <div className="space-y-1.5">
          <label
            htmlFor="onboarding-gender"
            className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
          >
            {t.onboarding.genderLabel}
          </label>
          <select
            id="onboarding-gender"
            name="gender"
            disabled={submitting}
            value={gender}
            onChange={(e) => setGender(e.target.value as Gender)}
            className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
          >
            <option value="male">{t.onboarding.genderMale}</option>
            <option value="female">{t.onboarding.genderFemale}</option>
          </select>
        </div>

        <div className="space-y-1.5">
          <label
            htmlFor="onboarding-activity"
            className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
          >
            {t.onboarding.activityLabel}
          </label>
          <select
            id="onboarding-activity"
            name="activity_level"
            disabled={submitting}
            value={activityLevel}
            onChange={(e) => setActivityLevel(e.target.value as ActivityLevel)}
            className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
          >
            <option value="light">{t.onboarding.activityLight}</option>
            <option value="moderate">{t.onboarding.activityModerate}</option>
            <option value="heavy">{t.onboarding.activityHeavy}</option>
          </select>
        </div>

        <button
          type="submit"
          disabled={submitting}
          className="w-full rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200 disabled:opacity-60"
        >
          {submitting ? t.common.saving : t.onboarding.submitButton}
        </button>
      </form>
    </div>
  );
}
