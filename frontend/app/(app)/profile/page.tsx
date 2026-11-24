"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PasswordInput } from "@/components/ui/PasswordInput";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
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

  if (
    isLoading ||
    !isAuthenticated ||
    !profile ||
    !profile.onboarding_completed
  ) {
    return (
      <Card className="rounded-3xl border border-border bg-card p-6 text-sm text-muted-foreground shadow-none ring-0">
        {t.common.loading}
      </Card>
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
      toast.success(t.profile.profileSaved);
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
      toast.success(t.profile.passwordChanged);
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

  const initials = (profile.display_name.trim() || profile.email || "N")
    .slice(0, 1)
    .toUpperCase();

  const genderLabels: Record<Gender, string> = {
    male: t.onboarding.genderMale,
    female: t.onboarding.genderFemale,
  };

  const activityOptions: { value: ActivityLevel; label: string }[] = [
    { value: "light", label: t.onboarding.activityLight },
    { value: "moderate", label: t.onboarding.activityModerate },
    { value: "heavy", label: t.onboarding.activityHeavy },
  ];

  const targetCards = [
    {
      label: t.profile.dailyCalories,
      value: `${profile.daily_calorie_target ?? "-"} ${t.profile.kcalUnit}`,
    },
    {
      label: t.profile.protein,
      value: `${profile.target_protein_g ?? "-"} ${t.profile.gramUnit}`,
    },
    {
      label: t.profile.carbs,
      value: `${profile.target_carbs_g ?? "-"} ${t.profile.gramUnit}`,
    },
    {
      label: t.profile.fat,
      value: `${profile.target_fat_g ?? "-"} ${t.profile.gramUnit}`,
    },
    {
      label: t.profile.fiber,
      value: `${profile.target_fiber_g ?? "-"} ${t.profile.gramUnit}`,
    },
  ];

  return (
    <div className="space-y-6">
      <h1 className="px-1 text-[22px] font-semibold tracking-[-0.015em] text-foreground sm:text-[26px]">
        {t.profile.title}
      </h1>

      <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5">
        <div className="flex flex-wrap items-center gap-4">
          <Avatar className="size-16">
            {isSafeAvatarUrl ? (
              <AvatarImage src={profile.profile_url} alt={t.profile.avatarAlt} />
            ) : null}
            <AvatarFallback className="text-lg font-semibold">
              {initials}
            </AvatarFallback>
          </Avatar>
          <div className="space-y-0.5">
            <h2 className="text-base font-semibold text-foreground">
              {profile.display_name.trim() || t.profile.summaryHeading}
            </h2>
            <p className="text-sm text-muted-foreground">{profile.email}</p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-5">
          {targetCards.map((item) => (
            <div key={item.label} className="rounded-2xl bg-muted p-3">
              <p className="text-xs text-muted-foreground">{item.label}</p>
              <p className="mt-1 text-base font-semibold tabular-nums text-foreground">
                {item.value}
              </p>
            </div>
          ))}
        </div>
      </Card>

      <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5">
        <h2 className="text-lg font-semibold text-foreground">
          {t.profile.editProfileHeading}
        </h2>

        {profileError ? <Alert variant="error">{profileError}</Alert> : null}
        {profileSuccess ? (
          <Alert variant="success">{profileSuccess}</Alert>
        ) : null}

        <form
          onSubmit={(e) => void handleProfileSubmit(e)}
          className="space-y-6"
        >
          <div className="space-y-1.5">
            <Label
              htmlFor="profile-display-name"
              className="text-sm font-medium text-foreground"
            >
              {t.onboarding.displayNameLabel}
            </Label>
            <Input
              id="profile-display-name"
              type="text"
              maxLength={50}
              autoComplete="nickname"
              disabled={profileSaving}
              value={form.displayName}
              onChange={(e) => updateField("displayName", e.target.value)}
              placeholder={t.onboarding.displayNamePlaceholder}
              className="h-11 max-w-sm rounded-2xl bg-card px-3.5 text-sm shadow-none"
            />
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div className="space-y-1.5">
              <Label
                htmlFor="profile-height"
                className="text-sm font-medium text-foreground"
              >
                {t.onboarding.heightLabel}
              </Label>
              <Input
                id="profile-height"
                type="number"
                min={100}
                max={250}
                step="0.1"
                required
                disabled={profileSaving}
                value={form.heightCm}
                onChange={(e) => updateField("heightCm", e.target.value)}
                className="h-11 rounded-2xl bg-card px-3.5 text-sm tabular-nums shadow-none"
              />
            </div>

            <div className="space-y-1.5">
              <Label
                htmlFor="profile-weight"
                className="text-sm font-medium text-foreground"
              >
                {t.onboarding.weightLabel}
              </Label>
              <Input
                id="profile-weight"
                type="number"
                min={25}
                max={300}
                step="0.1"
                required
                disabled={profileSaving}
                value={form.weightKg}
                onChange={(e) => updateField("weightKg", e.target.value)}
                className="h-11 rounded-2xl bg-card px-3.5 text-sm tabular-nums shadow-none"
              />
            </div>

            <div className="space-y-1.5">
              <Label
                htmlFor="profile-age"
                className="text-sm font-medium text-foreground"
              >
                {t.onboarding.ageLabel}
              </Label>
              <Input
                id="profile-age"
                type="number"
                min={13}
                max={100}
                step={1}
                required
                disabled={profileSaving}
                value={form.age}
                onChange={(e) => updateField("age", e.target.value)}
                className="h-11 rounded-2xl bg-card px-3.5 text-sm tabular-nums shadow-none"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label
                htmlFor="profile-gender"
                className="text-sm font-medium text-foreground"
              >
                {t.onboarding.genderLabel}
              </Label>
              <Select
                value={form.gender}
                disabled={profileSaving}
                onValueChange={(val) =>
                  updateField("gender", (val ?? "male") as Gender)
                }
              >
                <SelectTrigger
                  id="profile-gender"
                  className="h-11 w-full rounded-2xl border-border bg-card px-3.5 text-sm shadow-none"
                >
                  <SelectValue>{genderLabels[form.gender]}</SelectValue>
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

            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-foreground">
                {t.onboarding.activityLabel}
              </Label>
              <RadioGroup
                value={form.activityLevel}
                disabled={profileSaving}
                onValueChange={(val) =>
                  updateField("activityLevel", val as ActivityLevel)
                }
                className="grid grid-cols-3 gap-2"
              >
                {activityOptions.map((opt) => {
                  const selected = form.activityLevel === opt.value;
                  return (
                    <Label
                      key={opt.value}
                      htmlFor={`profile-activity-${opt.value}`}
                      className={`flex h-11 cursor-pointer items-center justify-center rounded-2xl border px-2 text-xs font-medium transition-colors duration-150 ${
                        selected
                          ? "border-[var(--brand-primary)] bg-[var(--brand-tint)] text-[var(--brand-primary)]"
                          : "border-border bg-card text-foreground hover:bg-accent"
                      }`}
                    >
                      <RadioGroupItem
                        id={`profile-activity-${opt.value}`}
                        value={opt.value}
                        className="sr-only"
                      />
                      <span>{opt.label}</span>
                    </Label>
                  );
                })}
              </RadioGroup>
            </div>
          </div>

          <Separator className="bg-border" />

          <div className="space-y-3">
            <h3 className="text-base font-semibold text-foreground">
              {t.profile.macroTargetsHeading}
            </h3>
            <p className="text-xs text-muted-foreground">
              {t.profile.recalcNote}
            </p>

            <div className="grid grid-cols-1 gap-3 sm:grid-cols-5">
              <div className="space-y-1.5">
                <Label
                  htmlFor="profile-calories"
                  className="text-xs font-medium text-foreground"
                >
                  {t.profile.dailyCalories} ({t.profile.kcalUnit})
                </Label>
                <Input
                  id="profile-calories"
                  type="number"
                  min={800}
                  max={6000}
                  step={1}
                  disabled={profileSaving}
                  value={form.dailyCalories}
                  onChange={(e) => updateField("dailyCalories", e.target.value)}
                  className="h-11 rounded-2xl bg-card px-3 text-sm tabular-nums shadow-none"
                />
              </div>

              <div className="space-y-1.5">
                <Label
                  htmlFor="profile-protein"
                  className="text-xs font-medium text-foreground"
                >
                  {t.profile.protein} ({t.profile.gramUnit})
                </Label>
                <Input
                  id="profile-protein"
                  type="number"
                  min={10}
                  max={500}
                  step="0.1"
                  disabled={profileSaving}
                  value={form.proteinG}
                  onChange={(e) => updateField("proteinG", e.target.value)}
                  className="h-11 rounded-2xl bg-card px-3 text-sm tabular-nums shadow-none"
                />
              </div>

              <div className="space-y-1.5">
                <Label
                  htmlFor="profile-carbs"
                  className="text-xs font-medium text-foreground"
                >
                  {t.profile.carbs} ({t.profile.gramUnit})
                </Label>
                <Input
                  id="profile-carbs"
                  type="number"
                  min={20}
                  max={1000}
                  step="0.1"
                  disabled={profileSaving}
                  value={form.carbsG}
                  onChange={(e) => updateField("carbsG", e.target.value)}
                  className="h-11 rounded-2xl bg-card px-3 text-sm tabular-nums shadow-none"
                />
              </div>

              <div className="space-y-1.5">
                <Label
                  htmlFor="profile-fat"
                  className="text-xs font-medium text-foreground"
                >
                  {t.profile.fat} ({t.profile.gramUnit})
                </Label>
                <Input
                  id="profile-fat"
                  type="number"
                  min={10}
                  max={300}
                  step="0.1"
                  disabled={profileSaving}
                  value={form.fatG}
                  onChange={(e) => updateField("fatG", e.target.value)}
                  className="h-11 rounded-2xl bg-card px-3 text-sm tabular-nums shadow-none"
                />
              </div>

              <div className="space-y-1.5">
                <Label
                  htmlFor="profile-fiber"
                  className="text-xs font-medium text-foreground"
                >
                  {t.profile.fiber} ({t.profile.gramUnit})
                </Label>
                <Input
                  id="profile-fiber"
                  type="number"
                  min={5}
                  max={150}
                  step="0.1"
                  disabled={profileSaving}
                  value={form.fiberG}
                  onChange={(e) => updateField("fiberG", e.target.value)}
                  className="h-11 rounded-2xl bg-card px-3 text-sm tabular-nums shadow-none"
                />
              </div>
            </div>
          </div>

          <Button
            type="submit"
            disabled={profileSaving}
            className="h-11 rounded-full px-6 text-sm font-medium shadow-none"
          >
            {profileSaving ? t.common.saving : t.profile.saveProfileButton}
          </Button>
        </form>
      </Card>

      <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-5">
        <h2 className="text-lg font-semibold text-foreground">
          {t.profile.changePasswordHeading}
        </h2>

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

          <Button
            type="submit"
            disabled={passwordSaving}
            className="h-11 rounded-full px-6 text-sm font-medium shadow-none"
          >
            {passwordSaving
              ? t.common.saving
              : t.profile.changePasswordButton}
          </Button>
        </form>
      </Card>

      <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-3">
        <h2 className="text-lg font-semibold text-foreground">
          {t.profile.privacyHeading}
        </h2>
        <div>
          <Link
            href="/privacy"
            className="inline-block text-sm font-medium text-foreground underline underline-offset-4 transition-colors duration-150 hover:text-primary"
          >
            {t.profile.privacyLinkText}
          </Link>
        </div>
      </Card>

      <Card className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0 gap-3">
        <h2 className="text-lg font-semibold text-foreground">
          {t.profile.deleteAccountHeading}
        </h2>
        <div>
          <Button
            type="button"
            variant="destructive"
            onClick={() => {
              setDeletePassword("");
              setDeleteError(null);
              setDeleteModalOpen(true);
            }}
            className="h-11 rounded-full bg-[var(--error-bg)] px-6 text-sm font-medium text-[var(--error-text)] hover:bg-[var(--error-bg)]/80 shadow-none"
          >
            {t.profile.deleteAccountButton}
          </Button>
        </div>
      </Card>

      <AlertDialog
        open={deleteModalOpen}
        onOpenChange={(open) => {
          if (!deleteSubmitting) {
            setDeleteModalOpen(open);
          }
        }}
      >
        <AlertDialogContent className="max-w-md rounded-3xl border border-border bg-card p-6 shadow-none ring-0">
          <AlertDialogHeader>
            <AlertDialogTitle>{t.profile.deleteDialogTitle}</AlertDialogTitle>
            <AlertDialogDescription>
              {t.profile.deleteDialogQuestion} {t.profile.deleteDialogWarning}
            </AlertDialogDescription>
          </AlertDialogHeader>

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

            <AlertDialogFooter className="mx-0 mb-0 rounded-none border-0 bg-transparent p-0 pt-2">
              <AlertDialogCancel
                disabled={deleteSubmitting}
                className="h-10 rounded-full px-5 shadow-none"
              >
                {t.common.cancel}
              </AlertDialogCancel>
              <Button
                type="submit"
                variant="destructive"
                disabled={deleteSubmitting}
                className="h-10 rounded-full bg-[var(--error-bg)] px-5 text-sm font-medium text-[var(--error-text)] hover:bg-[var(--error-bg)]/80 shadow-none"
              >
                {deleteSubmitting
                  ? t.profile.deleteDialogDeleting
                  : t.profile.deleteDialogConfirm}
              </Button>
            </AlertDialogFooter>
          </form>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
