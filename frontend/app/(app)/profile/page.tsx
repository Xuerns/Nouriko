"use client";

import { useEffect, useState, type FormEvent } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { PasswordInput } from "@/components/ui/PasswordInput";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";
import type {
  ActivityLevel,
  Gender,
  ProfileDTO,
  UpdateProfilePayload,
} from "@/lib/types";

interface ProfileFormState {
  displayName: string;
  heightCm: string;
  weightKg: string;
  age: string;
  gender: Gender;
  activityLevel: ActivityLevel;
  dailyCalories: string;
  proteinG: string;
  carbsG: string;
  fatG: string;
  fiberG: string;
}

function profileToFormState(p: ProfileDTO): ProfileFormState {
  return {
    displayName: p.display_name ?? "",
    heightCm: p.height_cm !== null ? String(p.height_cm) : "",
    weightKg: p.weight_kg !== null ? String(p.weight_kg) : "",
    age: p.age !== null ? String(p.age) : "",
    gender: p.gender ?? "male",
    activityLevel: p.activity_level ?? "moderate",
    dailyCalories:
      p.daily_calorie_target !== null ? String(p.daily_calorie_target) : "",
    proteinG: p.target_protein_g !== null ? String(p.target_protein_g) : "",
    carbsG: p.target_carbs_g !== null ? String(p.target_carbs_g) : "",
    fatG: p.target_fat_g !== null ? String(p.target_fat_g) : "",
    fiberG: p.target_fiber_g !== null ? String(p.target_fiber_g) : "",
  };
}

export default function ProfilePage() {
  const router = useRouter();
  const {
    token,
    profile,
    isAuthenticated,
    isLoading,
    setProfile,
    clearSession,
    handleUnauthorized,
  } = useAuth();
  const { language, t } = useI18n();

  const [formOverride, setFormOverride] = useState<ProfileFormState | null>(
    null
  );
  const [profileSaving, setProfileSaving] = useState(false);
  const [profileError, setProfileError] = useState<string | null>(null);
  const [profileSuccess, setProfileSuccess] = useState<string | null>(null);

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmNewPassword, setConfirmNewPassword] = useState("");
  const [passwordSaving, setPasswordSaving] = useState(false);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState<string | null>(null);

  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleteSubmitting, setDeleteSubmitting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated) {
      router.replace("/login");
      return;
    }
    if (profile && !profile.onboarding_completed) {
      router.replace("/onboarding");
    }
  }, [isAuthenticated, isLoading, profile, router]);

  useEffect(() => {
    if (!deleteModalOpen) return;
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape" && !deleteSubmitting) {
        setDeleteModalOpen(false);
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [deleteModalOpen, deleteSubmitting]);

  if (isLoading || !isAuthenticated || !profile || !profile.onboarding_completed) {
    return (
      <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
        {t.common.loading}
      </div>
    );
  }

  const baseForm = profileToFormState(profile);
  const form = formOverride ?? baseForm;

  function updateField<K extends keyof ProfileFormState>(
    key: K,
    value: ProfileFormState[K]
  ) {
    setFormOverride((prev) => ({
      ...(prev ?? baseForm),
      [key]: value,
    }));
  }

  async function handleProfileSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!token || !profile) return;
    setProfileError(null);
    setProfileSuccess(null);

    const payload: UpdateProfilePayload = {};

    const parsedHeight = Number(form.heightCm);
    const parsedWeight = Number(form.weightKg);
    const parsedAge = Number(form.age);

    if (
      !Number.isFinite(parsedHeight) ||
      !Number.isFinite(parsedWeight) ||
      !Number.isInteger(parsedAge)
    ) {
      setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
      return;
    }

    const trimmedDisplayName = form.displayName.trim();
    if (trimmedDisplayName !== (profile.display_name ?? "")) {
      payload.display_name = trimmedDisplayName;
    }
    if (parsedHeight !== profile.height_cm) payload.height_cm = parsedHeight;
    if (parsedWeight !== profile.weight_kg) payload.weight_kg = parsedWeight;
    if (parsedAge !== profile.age) payload.age = parsedAge;
    if (form.gender !== profile.gender) payload.gender = form.gender;
    if (form.activityLevel !== profile.activity_level) {
      payload.activity_level = form.activityLevel;
    }

    // Only include explicit macro/calorie target overrides when the user edited them,
    // so changing body metrics alone triggers automatic Mifflin-St Jeor recalculation.
    const parsedCal = Number(form.dailyCalories);
    const parsedPro = Number(form.proteinG);
    const parsedCarbs = Number(form.carbsG);
    const parsedFat = Number(form.fatG);
    const parsedFiber = Number(form.fiberG);

    if (form.dailyCalories !== baseForm.dailyCalories) {
      if (!Number.isInteger(parsedCal)) {
        setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
        return;
      }
      payload.daily_calorie_target = parsedCal;
    }
    if (form.proteinG !== baseForm.proteinG) {
      if (!Number.isFinite(parsedPro)) {
        setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
        return;
      }
      payload.target_protein_g = parsedPro;
    }
    if (form.carbsG !== baseForm.carbsG) {
      if (!Number.isFinite(parsedCarbs)) {
        setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
        return;
      }
      payload.target_carbs_g = parsedCarbs;
    }
    if (form.fatG !== baseForm.fatG) {
      if (!Number.isFinite(parsedFat)) {
        setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
        return;
      }
      payload.target_fat_g = parsedFat;
    }
    if (form.fiberG !== baseForm.fiberG) {
      if (!Number.isFinite(parsedFiber)) {
        setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
        return;
      }
      payload.target_fiber_g = parsedFiber;
    }

    if (Object.keys(payload).length === 0) {
      setProfileSuccess(t.profile.profileSaved);
      return;
    }

    setProfileSaving(true);
    try {
      const res = await apiClient.updateProfile(token, payload);
      setProfile(res.profile);
      setFormOverride(null);
      setProfileSuccess(t.profile.profileSaved);
    } catch (err) {
      if (handleUnauthorized(err)) return;
      const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
      setProfileError(getFriendlyErrorMessage(code, language));
    } finally {
      setProfileSaving(false);
    }
  }

  async function handlePasswordSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!token) return;
    setPasswordError(null);
    setPasswordSuccess(null);

    if (newPassword !== confirmNewPassword) {
      setPasswordError(t.common.passwordMismatch);
      return;
    }

    setPasswordSaving(true);
    try {
      await apiClient.changePassword(token, {
        current_password: currentPassword,
        new_password: newPassword,
      });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmNewPassword("");
      setPasswordSuccess(t.profile.passwordChanged);
    } catch (err) {
      if (handleUnauthorized(err)) return;
      const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
      setPasswordError(getFriendlyErrorMessage(code, language));
    } finally {
      setPasswordSaving(false);
    }
  }

  async function handleDeleteAccount(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!token) return;
    setDeleteError(null);
    setDeleteSubmitting(true);

    try {
      await apiClient.deleteAccount(token, deletePassword);
      clearSession(false);
      router.replace("/");
    } catch (err) {
      if (handleUnauthorized(err)) return;
      const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
      setDeleteError(getFriendlyErrorMessage(code, language));
    } finally {
      setDeleteSubmitting(false);
    }
  }

  const isSafeAvatarUrl =
    typeof profile.profile_url === "string" &&
    profile.profile_url.startsWith("https://api.dicebear.com/");

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-semibold tracking-tight text-neutral-900 dark:text-neutral-100">
        {t.profile.title}
      </h1>

      <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-5">
        <div className="flex flex-wrap items-center gap-4">
          {isSafeAvatarUrl ? (
            <Image
              src={profile.profile_url}
              alt={t.profile.avatarAlt}
              width={64}
              height={64}
              unoptimized
              className="h-16 w-16 rounded-lg border border-neutral-300 dark:border-neutral-700"
            />
          ) : null}
          <div className="space-y-1">
            <h2 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
              {profile.display_name.trim() || t.profile.summaryHeading}
            </h2>
            <p className="text-sm text-neutral-600 dark:text-neutral-400">
              {profile.email}
            </p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
          <div className="rounded-md border border-neutral-300 dark:border-neutral-700 p-3">
            <p className="text-xs text-neutral-600 dark:text-neutral-400">
              {t.profile.dailyCalories}
            </p>
            <p className="mt-1 font-mono text-base font-semibold text-neutral-900 dark:text-neutral-100">
              {profile.daily_calorie_target ?? "-"} {t.profile.kcalUnit}
            </p>
          </div>
          <div className="rounded-md border border-neutral-300 dark:border-neutral-700 p-3">
            <p className="text-xs text-neutral-600 dark:text-neutral-400">
              {t.profile.protein}
            </p>
            <p className="mt-1 font-mono text-base font-semibold text-neutral-900 dark:text-neutral-100">
              {profile.target_protein_g ?? "-"} {t.profile.gramUnit}
            </p>
          </div>
          <div className="rounded-md border border-neutral-300 dark:border-neutral-700 p-3">
            <p className="text-xs text-neutral-600 dark:text-neutral-400">
              {t.profile.carbs}
            </p>
            <p className="mt-1 font-mono text-base font-semibold text-neutral-900 dark:text-neutral-100">
              {profile.target_carbs_g ?? "-"} {t.profile.gramUnit}
            </p>
          </div>
          <div className="rounded-md border border-neutral-300 dark:border-neutral-700 p-3">
            <p className="text-xs text-neutral-600 dark:text-neutral-400">
              {t.profile.fat}
            </p>
            <p className="mt-1 font-mono text-base font-semibold text-neutral-900 dark:text-neutral-100">
              {profile.target_fat_g ?? "-"} {t.profile.gramUnit}
            </p>
          </div>
          <div className="rounded-md border border-neutral-300 dark:border-neutral-700 p-3">
            <p className="text-xs text-neutral-600 dark:text-neutral-400">
              {t.profile.fiber}
            </p>
            <p className="mt-1 font-mono text-base font-semibold text-neutral-900 dark:text-neutral-100">
              {profile.target_fiber_g ?? "-"} {t.profile.gramUnit}
            </p>
          </div>
        </div>
      </section>

      <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-5">
        <h2 className="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
          {t.profile.editProfileHeading}
        </h2>

        {profileError ? <Alert variant="error">{profileError}</Alert> : null}
        {profileSuccess ? (
          <Alert variant="success">{profileSuccess}</Alert>
        ) : null}

        <form onSubmit={(e) => void handleProfileSubmit(e)} className="space-y-6">
          <div className="space-y-1.5">
            <label
              htmlFor="profile-display-name"
              className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
            >
              {t.onboarding.displayNameLabel}
            </label>
            <input
              id="profile-display-name"
              type="text"
              maxLength={50}
              autoComplete="nickname"
              disabled={profileSaving}
              value={form.displayName}
              onChange={(e) => updateField("displayName", e.target.value)}
              placeholder={t.onboarding.displayNamePlaceholder}
              className="w-full max-w-sm rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
            />
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div className="space-y-1.5">
              <label
                htmlFor="profile-height"
                className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
              >
                {t.onboarding.heightLabel}
              </label>
              <input
                id="profile-height"
                type="number"
                min={100}
                max={250}
                step="0.1"
                required
                disabled={profileSaving}
                value={form.heightCm}
                onChange={(e) => updateField("heightCm", e.target.value)}
                className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
              />
            </div>

            <div className="space-y-1.5">
              <label
                htmlFor="profile-weight"
                className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
              >
                {t.onboarding.weightLabel}
              </label>
              <input
                id="profile-weight"
                type="number"
                min={25}
                max={300}
                step="0.1"
                required
                disabled={profileSaving}
                value={form.weightKg}
                onChange={(e) => updateField("weightKg", e.target.value)}
                className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
              />
            </div>

            <div className="space-y-1.5">
              <label
                htmlFor="profile-age"
                className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
              >
                {t.onboarding.ageLabel}
              </label>
              <input
                id="profile-age"
                type="number"
                min={13}
                max={100}
                step={1}
                required
                disabled={profileSaving}
                value={form.age}
                onChange={(e) => updateField("age", e.target.value)}
                className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <label
                htmlFor="profile-gender"
                className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
              >
                {t.onboarding.genderLabel}
              </label>
              <select
                id="profile-gender"
                disabled={profileSaving}
                value={form.gender}
                onChange={(e) =>
                  updateField("gender", e.target.value as Gender)
                }
                className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
              >
                <option value="male">{t.onboarding.genderMale}</option>
                <option value="female">{t.onboarding.genderFemale}</option>
              </select>
            </div>

            <div className="space-y-1.5">
              <label
                htmlFor="profile-activity"
                className="block text-sm font-medium text-neutral-900 dark:text-neutral-100"
              >
                {t.onboarding.activityLabel}
              </label>
              <select
                id="profile-activity"
                disabled={profileSaving}
                value={form.activityLevel}
                onChange={(e) =>
                  updateField("activityLevel", e.target.value as ActivityLevel)
                }
                className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
              >
                <option value="light">{t.onboarding.activityLight}</option>
                <option value="moderate">{t.onboarding.activityModerate}</option>
                <option value="heavy">{t.onboarding.activityHeavy}</option>
              </select>
            </div>
          </div>

          <div className="space-y-3 border-t border-neutral-300 dark:border-neutral-700 pt-5">
            <h3 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
              {t.profile.macroTargetsHeading}
            </h3>
            <p className="text-xs text-neutral-600 dark:text-neutral-400">
              {t.profile.recalcNote}
            </p>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-5">
              <div className="space-y-1.5">
                <label
                  htmlFor="profile-calories"
                  className="block text-xs font-medium text-neutral-900 dark:text-neutral-100"
                >
                  {t.profile.dailyCalories} ({t.profile.kcalUnit})
                </label>
                <input
                  id="profile-calories"
                  type="number"
                  min={800}
                  max={6000}
                  step={1}
                  disabled={profileSaving}
                  value={form.dailyCalories}
                  onChange={(e) => updateField("dailyCalories", e.target.value)}
                  className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
                />
              </div>

              <div className="space-y-1.5">
                <label
                  htmlFor="profile-protein"
                  className="block text-xs font-medium text-neutral-900 dark:text-neutral-100"
                >
                  {t.profile.protein} ({t.profile.gramUnit})
                </label>
                <input
                  id="profile-protein"
                  type="number"
                  min={10}
                  max={500}
                  step="0.1"
                  disabled={profileSaving}
                  value={form.proteinG}
                  onChange={(e) => updateField("proteinG", e.target.value)}
                  className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
                />
              </div>

              <div className="space-y-1.5">
                <label
                  htmlFor="profile-carbs"
                  className="block text-xs font-medium text-neutral-900 dark:text-neutral-100"
                >
                  {t.profile.carbs} ({t.profile.gramUnit})
                </label>
                <input
                  id="profile-carbs"
                  type="number"
                  min={20}
                  max={1000}
                  step="0.1"
                  disabled={profileSaving}
                  value={form.carbsG}
                  onChange={(e) => updateField("carbsG", e.target.value)}
                  className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
                />
              </div>

              <div className="space-y-1.5">
                <label
                  htmlFor="profile-fat"
                  className="block text-xs font-medium text-neutral-900 dark:text-neutral-100"
                >
                  {t.profile.fat} ({t.profile.gramUnit})
                </label>
                <input
                  id="profile-fat"
                  type="number"
                  min={10}
                  max={300}
                  step="0.1"
                  disabled={profileSaving}
                  value={form.fatG}
                  onChange={(e) => updateField("fatG", e.target.value)}
                  className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
                />
              </div>

              <div className="space-y-1.5">
                <label
                  htmlFor="profile-fiber"
                  className="block text-xs font-medium text-neutral-900 dark:text-neutral-100"
                >
                  {t.profile.fiber} ({t.profile.gramUnit})
                </label>
                <input
                  id="profile-fiber"
                  type="number"
                  min={5}
                  max={150}
                  step="0.1"
                  disabled={profileSaving}
                  value={form.fiberG}
                  onChange={(e) => updateField("fiberG", e.target.value)}
                  className="w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 focus:outline-none focus:ring-2 focus:ring-neutral-900 dark:focus:ring-neutral-100 disabled:opacity-60"
                />
              </div>
            </div>
          </div>

          <button
            type="submit"
            disabled={profileSaving}
            className="rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200 disabled:opacity-60"
          >
            {profileSaving ? t.common.saving : t.profile.saveProfileButton}
          </button>
        </form>
      </section>

      <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-5">
        <div className="space-y-1">
          <h2 className="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
            {t.profile.changePasswordHeading}
          </h2>
          <p className="text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed">
            {t.profile.changePasswordDescription}
          </p>
        </div>

        {passwordError ? <Alert variant="error">{passwordError}</Alert> : null}
        {passwordSuccess ? (
          <Alert variant="success">{passwordSuccess}</Alert>
        ) : null}

        <form
          onSubmit={(e) => void handlePasswordSubmit(e)}
          className="max-w-md space-y-4"
        >
          <PasswordInput
            id="profile-current-password"
            name="currentPassword"
            label={t.common.currentPasswordLabel}
            value={currentPassword}
            onChange={setCurrentPassword}
            autoComplete="current-password"
            disabled={passwordSaving}
          />

          <PasswordInput
            id="profile-new-password"
            name="newPassword"
            label={t.common.newPasswordLabel}
            value={newPassword}
            onChange={setNewPassword}
            autoComplete="new-password"
            minLength={8}
            disabled={passwordSaving}
            showStrength
            hint={t.common.passwordMinHint}
          />

          <PasswordInput
            id="profile-confirm-new-password"
            name="confirmNewPassword"
            label={t.common.confirmPasswordLabel}
            value={confirmNewPassword}
            onChange={setConfirmNewPassword}
            autoComplete="new-password"
            minLength={8}
            disabled={passwordSaving}
          />

          <button
            type="submit"
            disabled={passwordSaving}
            className="rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-800 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-200 disabled:opacity-60"
          >
            {passwordSaving
              ? t.common.saving
              : t.profile.changePasswordButton}
          </button>
        </form>
      </section>

      <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-3">
        <div className="space-y-1">
          <h2 className="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
            {t.profile.privacyHeading}
          </h2>
          <p className="text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed">
            {t.profile.privacyDescription}
          </p>
        </div>
        <Link
          href="/privacy"
          className="inline-block text-sm font-medium text-neutral-900 dark:text-neutral-100 underline underline-offset-4"
        >
          {t.profile.privacyLinkText}
        </Link>
      </section>

      <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 space-y-3">
        <div className="space-y-1">
          <h2 className="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
            {t.profile.deleteAccountHeading}
          </h2>
          <p className="text-sm text-neutral-600 dark:text-neutral-400 leading-relaxed">
            {t.profile.deleteAccountDescription}
          </p>
        </div>
        <button
          type="button"
          onClick={() => {
            setDeletePassword("");
            setDeleteError(null);
            setDeleteModalOpen(true);
          }}
          className="rounded-md bg-red-700 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-red-800 dark:bg-red-600 dark:hover:bg-red-700"
        >
          {t.profile.deleteAccountButton}
        </button>
      </section>

      {deleteModalOpen ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/60 p-4"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-account-dialog-title"
        >
          <div className="w-full max-w-md rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 p-6 space-y-4">
            <h2
              id="delete-account-dialog-title"
              className="text-lg font-semibold text-neutral-900 dark:text-neutral-100"
            >
              {t.profile.deleteDialogTitle}
            </h2>

            <p className="text-sm font-medium text-neutral-900 dark:text-neutral-100">
              {t.profile.deleteDialogQuestion}
            </p>
            <p className="text-sm text-neutral-700 dark:text-neutral-300">
              {t.profile.deleteDialogWarning}
            </p>

            {deleteError ? <Alert variant="error">{deleteError}</Alert> : null}

            <form
              onSubmit={(e) => void handleDeleteAccount(e)}
              className="space-y-4"
            >
              <PasswordInput
                id="delete-account-password"
                name="deletePassword"
                label={t.profile.deleteDialogPasswordLabel}
                value={deletePassword}
                onChange={setDeletePassword}
                autoComplete="current-password"
                disabled={deleteSubmitting}
              />

              <div className="flex flex-wrap justify-end gap-2 pt-2">
                <button
                  type="button"
                  disabled={deleteSubmitting}
                  onClick={() => setDeleteModalOpen(false)}
                  className="rounded-md border border-neutral-300 dark:border-neutral-700 px-4 py-2 text-sm font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-60"
                >
                  {t.common.cancel}
                </button>
                <button
                  type="submit"
                  disabled={deleteSubmitting}
                  className="rounded-md bg-red-700 px-4 py-2 text-sm font-medium text-white transition-colors duration-150 hover:bg-red-800 dark:bg-red-600 dark:hover:bg-red-700 disabled:opacity-60"
                >
                  {deleteSubmitting
                    ? t.profile.deleteDialogDeleting
                    : t.profile.deleteDialogConfirm}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  );
}
