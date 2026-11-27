"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ArrowUpRight,
  Beef,
  Droplet,
  Flame,
  KeyRound,
  Leaf,
  RotateCcw,
  Shield,
  SlidersHorizontal,
  Trash2,
  Wheat,
} from "lucide-react";
import { toast } from "sonner";
import { SocialAuthButtons } from "@/components/auth/SocialAuthButtons";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ProfilePageSkeleton } from "@/components/ui/PageSkeletons";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
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

function sanitizeDecimalInput(raw: string): string {
  const normalized = raw.replace(",", ".").replace(/[^0-9.]/g, "");
  const dotIdx = normalized.indexOf(".");
  if (dotIdx === -1) return normalized;
  return (
    normalized.slice(0, dotIdx + 1) +
    normalized.slice(dotIdx + 1).replace(/\./g, "")
  );
}

function sanitizeIntInput(raw: string): string {
  return raw.replace(/\D/g, "");
}

function parseFiniteInRange(raw: string, min: number, max: number): number | null {
  const trimmed = raw.trim();
  if (!trimmed) return null;
  const num = Number(trimmed);
  if (!Number.isFinite(num) || num < min || num > max) return null;
  return num;
}

function parseIntInRange(raw: string, min: number, max: number): number | null {
  const trimmed = raw.trim();
  if (!trimmed) return null;
  const num = Number(trimmed);
  if (!Number.isInteger(num) || num < min || num > max) return null;
  return num;
}

function hasFormChanges(a: ProfileFormState, b: ProfileFormState): boolean {
  return (
    a.displayName.trim() !== b.displayName.trim() ||
    a.heightCm !== b.heightCm ||
    a.weightKg !== b.weightKg ||
    a.age !== b.age ||
    a.gender !== b.gender ||
    a.activityLevel !== b.activityLevel ||
    a.dailyCalories !== b.dailyCalories ||
    a.proteinG !== b.proteinG ||
    a.carbsG !== b.carbsG ||
    a.fatG !== b.fatG ||
    a.fiberG !== b.fiberG
  );
}

function ActivityLevelBars({ level }: { level: 1 | 2 | 3 }) {
  return (
    <svg
      viewBox="0 0 16 14"
      className="size-3.5 shrink-0"
      aria-hidden="true"
      fill="currentColor"
    >
      <rect x="1" y="8" width="3.2" height="5" rx="1.2" />
      <rect
        x="6.4"
        y="4.5"
        width="3.2"
        height="8.5"
        rx="1.2"
        className={level >= 2 ? "opacity-100" : "opacity-25"}
      />
      <rect
        x="11.8"
        y="1"
        width="3.2"
        height="12"
        rx="1.2"
        className={level >= 3 ? "opacity-100" : "opacity-25"}
      />
    </svg>
  );
}

interface UnitNumberFieldProps {
  id: string;
  label: string;
  unit: string;
  value: string;
  disabled?: boolean;
  integer?: boolean;
  onChange: (value: string) => void;
}

function UnitNumberField({
  id,
  label,
  unit,
  value,
  disabled = false,
  integer = false,
  onChange,
}: UnitNumberFieldProps) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-xs font-medium text-foreground">
        {label}
      </Label>
      <div className="relative">
        <Input
          id={id}
          type="text"
          inputMode={integer ? "numeric" : "decimal"}
          autoComplete="off"
          disabled={disabled}
          value={value}
          onChange={(e) =>
            onChange(
              integer
                ? sanitizeIntInput(e.target.value)
                : sanitizeDecimalInput(e.target.value)
            )
          }
          className="h-10 rounded-2xl bg-card pr-11 pl-3 text-sm tabular-nums shadow-none"
        />
        <span
          aria-hidden="true"
          className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs text-muted-foreground"
        >
          {unit}
        </span>
      </div>
    </div>
  );
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

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmNewPassword, setConfirmNewPassword] = useState("");
  const [passwordSaving, setPasswordSaving] = useState(false);
  const [passwordError, setPasswordError] = useState<string | null>(null);

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
    return <ProfilePageSkeleton label={t.common.loading} />;
  }

  const baseForm = profileToFormState(profile);
  const form = formOverride ?? baseForm;
  const dirty = hasFormChanges(form, baseForm);
  const hasPassword = profile.has_password !== false;

  function updateField<K extends keyof ProfileFormState>(
    key: K,
    value: ProfileFormState[K]
  ) {
    setProfileError(null);
    setFormOverride((prev) => ({
      ...(prev ?? baseForm),
      [key]: value,
    }));
  }

  function handleResetForm() {
    setFormOverride(null);
    setProfileError(null);
  }

  async function handleProfileSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!token || !profile) return;
    setProfileError(null);

    const parsedHeight = parseFiniteInRange(form.heightCm, 100, 250);
    const parsedWeight = parseFiniteInRange(form.weightKg, 25, 300);
    const parsedAge = parseIntInRange(form.age, 13, 100);

    if (parsedHeight === null || parsedWeight === null || parsedAge === null) {
      setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
      return;
    }

    const payload: UpdateProfilePayload = {};
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

    if (form.dailyCalories !== baseForm.dailyCalories) {
      const parsedCal = parseIntInRange(form.dailyCalories, 800, 6000);
      if (parsedCal === null) {
        setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
        return;
      }
      payload.daily_calorie_target = parsedCal;
    }
    if (form.proteinG !== baseForm.proteinG) {
      const parsedPro = parseFiniteInRange(form.proteinG, 10, 500);
      if (parsedPro === null) {
        setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
        return;
      }
      payload.target_protein_g = parsedPro;
    }
    if (form.carbsG !== baseForm.carbsG) {
      const parsedCarbs = parseFiniteInRange(form.carbsG, 20, 1000);
      if (parsedCarbs === null) {
        setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
        return;
      }
      payload.target_carbs_g = parsedCarbs;
    }
    if (form.fatG !== baseForm.fatG) {
      const parsedFat = parseFiniteInRange(form.fatG, 10, 300);
      if (parsedFat === null) {
        setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
        return;
      }
      payload.target_fat_g = parsedFat;
    }
    if (form.fiberG !== baseForm.fiberG) {
      const parsedFiber = parseFiniteInRange(form.fiberG, 5, 150);
      if (parsedFiber === null) {
        setProfileError(getFriendlyErrorMessage("INVALID_INPUT", language));
        return;
      }
      payload.target_fiber_g = parsedFiber;
    }

    if (Object.keys(payload).length === 0) {
      toast.success(t.profile.profileSaved);
      return;
    }

    setProfileSaving(true);
    try {
      const res = await apiClient.updateProfile(token, payload);
      setProfile(res.profile);
      setFormOverride(null);
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

    if (!currentPassword || !newPassword || !confirmNewPassword) {
      setPasswordError(getFriendlyErrorMessage("INVALID_INPUT", language));
      return;
    }

    if (newPassword.length < 8) {
      setPasswordError(t.common.passwordMinHint);
      return;
    }

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

    if (!deletePassword) {
      setDeleteError(getFriendlyErrorMessage("INVALID_INPUT", language));
      return;
    }

    setDeleteSubmitting(true);
    try {
      await apiClient.deleteAccount(token, { password: deletePassword });
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

  async function handleGoogleDeleteAccount({
    idToken,
    nonce,
  }: {
    idToken: string;
    nonce: string;
  }) {
    if (!token) return;
    setDeleteError(null);
    setDeleteSubmitting(true);
    try {
      await apiClient.deleteAccount(token, {
        google_id_token: idToken,
        nonce,
      });
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

  const activityLabels: Record<ActivityLevel, string> = {
    light: t.onboarding.activityLight,
    moderate: t.onboarding.activityModerate,
    heavy: t.onboarding.activityHeavy,
  };

  const activityOptions: {
    value: ActivityLevel;
    label: string;
    hint: string;
    level: 1 | 2 | 3;
  }[] = [
    {
      value: "light",
      label: t.onboarding.activityLight,
      hint: t.profile.activityLightHint,
      level: 1,
    },
    {
      value: "moderate",
      label: t.onboarding.activityModerate,
      hint: t.profile.activityModerateHint,
      level: 2,
    },
    {
      value: "heavy",
      label: t.onboarding.activityHeavy,
      hint: t.profile.activityHeavyHint,
      level: 3,
    },
  ];

  const targetCards = [
    {
      label: t.profile.dailyCalories,
      value: `${profile.daily_calorie_target ?? "-"} ${t.profile.kcalUnit}`,
      accent: "text-[var(--macro-calories)]",
      icon: Flame,
    },
    {
      label: t.profile.protein,
      value: `${profile.target_protein_g ?? "-"} ${t.profile.gramUnit}`,
      accent: "text-[var(--macro-protein)]",
      icon: Beef,
    },
    {
      label: t.profile.carbs,
      value: `${profile.target_carbs_g ?? "-"} ${t.profile.gramUnit}`,
      accent: "text-[var(--macro-carbs)]",
      icon: Wheat,
    },
    {
      label: t.profile.fat,
      value: `${profile.target_fat_g ?? "-"} ${t.profile.gramUnit}`,
      accent: "text-[var(--macro-fat)]",
      icon: Droplet,
    },
    {
      label: t.profile.fiber,
      value: `${profile.target_fiber_g ?? "-"} ${t.profile.gramUnit}`,
      accent: "text-[var(--macro-fiber)]",
      icon: Leaf,
    },
  ];

  const quickMeta: string[] = [];
  if (profile.height_cm !== null) {
    quickMeta.push(`${profile.height_cm} ${t.profile.cmUnit}`);
  }
  if (profile.weight_kg !== null) {
    quickMeta.push(`${profile.weight_kg} ${t.profile.kgUnit}`);
  }
  if (profile.age !== null) {
    quickMeta.push(`${profile.age} ${t.profile.ageUnit}`);
  }

  return (
    <div className="space-y-5">
      <div className="space-y-1 px-0.5">
        <h1 className="text-[22px] font-semibold tracking-[-0.015em] text-foreground sm:text-[26px]">
          {t.profile.title}
        </h1>
        <p className="text-sm text-muted-foreground">{t.profile.subtitle}</p>
      </div>

      <Card className="rounded-3xl border border-border bg-card py-4 shadow-none ring-0 gap-4">
        <CardContent className="space-y-4 px-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <Avatar className="size-12 shrink-0">
                {isSafeAvatarUrl ? (
                  <AvatarImage
                    src={profile.profile_url}
                    alt={t.profile.avatarAlt}
                  />
                ) : null}
                <AvatarFallback className="text-base font-semibold">
                  {initials}
                </AvatarFallback>
              </Avatar>
              <div className="min-w-0 space-y-0.5">
                <h2 className="truncate text-base font-semibold text-foreground">
                  {profile.display_name.trim() || t.profile.summaryHeading}
                </h2>
                <p className="truncate text-xs text-muted-foreground">
                  {profile.email}
                </p>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-1.5">
              {quickMeta.length > 0 ? (
                <Badge
                  variant="secondary"
                  className="h-7 rounded-full border-0 bg-muted px-3 text-xs font-medium tabular-nums text-muted-foreground"
                >
                  {quickMeta.join(" · ")}
                </Badge>
              ) : null}
              {profile.activity_level ? (
                <Badge
                  variant="secondary"
                  className="h-7 rounded-full border-0 bg-[var(--brand-tint)] px-3 text-xs font-medium text-[var(--brand-primary)]"
                >
                  {activityLabels[profile.activity_level]}
                </Badge>
              ) : null}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
            {targetCards.map((item, index) => {
              const Icon = item.icon;
              return (
                <div
                  key={item.label}
                  className={`rounded-2xl bg-muted px-3 py-2.5 ${
                    index === 0 ? "col-span-2 sm:col-span-1" : ""
                  }`}
                >
                  <div
                    className={`flex items-center justify-between gap-1.5 ${item.accent}`}
                  >
                    <p className="truncate text-[11px] font-medium">
                      {item.label}
                    </p>
                    <Icon
                      className="size-3.5 shrink-0 stroke-[1.75]"
                      aria-hidden="true"
                    />
                  </div>
                  <p className="mt-1 text-sm font-semibold tabular-nums text-foreground">
                    {item.value}
                  </p>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="body" className="gap-4">
        <TabsList
          aria-label={t.profile.tabsAriaLabel}
          className="no-scrollbar flex h-auto w-full flex-nowrap justify-start gap-1.5 overflow-x-auto overflow-y-hidden rounded-full border border-border bg-card p-1 group-data-horizontal/tabs:h-auto sm:w-fit"
        >
          <TabsTrigger
            value="body"
            className="h-9 flex-none rounded-full border border-transparent px-4 text-xs font-medium text-muted-foreground shadow-none after:hidden data-active:border-[var(--brand-primary)]/30 data-active:bg-[var(--brand-tint)] data-active:text-[var(--brand-primary)] data-active:shadow-none dark:data-active:border-[var(--brand-primary)]/30 dark:data-active:bg-[var(--brand-tint)] dark:data-active:text-[var(--brand-primary)]"
          >
            <SlidersHorizontal
              className="size-3.5 stroke-[1.75]"
              aria-hidden="true"
            />
            <span>{t.profile.tabBodyTargets}</span>
          </TabsTrigger>
          <TabsTrigger
            value="security"
            className="h-9 flex-none rounded-full border border-transparent px-4 text-xs font-medium text-muted-foreground shadow-none after:hidden data-active:border-[var(--brand-primary)]/30 data-active:bg-[var(--brand-tint)] data-active:text-[var(--brand-primary)] data-active:shadow-none dark:data-active:border-[var(--brand-primary)]/30 dark:data-active:bg-[var(--brand-tint)] dark:data-active:text-[var(--brand-primary)]"
          >
            <KeyRound className="size-3.5 stroke-[1.75]" aria-hidden="true" />
            <span>{t.profile.tabSecurity}</span>
          </TabsTrigger>
          <TabsTrigger
            value="account"
            className="h-9 flex-none rounded-full border border-transparent px-4 text-xs font-medium text-muted-foreground shadow-none after:hidden data-active:border-[var(--brand-primary)]/30 data-active:bg-[var(--brand-tint)] data-active:text-[var(--brand-primary)] data-active:shadow-none dark:data-active:border-[var(--brand-primary)]/30 dark:data-active:bg-[var(--brand-tint)] dark:data-active:text-[var(--brand-primary)]"
          >
            <Shield className="size-3.5 stroke-[1.75]" aria-hidden="true" />
            <span>{t.profile.tabAccount}</span>
          </TabsTrigger>
        </TabsList>

        <TabsContent value="body">
          <Card className="rounded-3xl border border-border bg-card py-0 shadow-none ring-0 gap-0">
            <form
              noValidate
              onSubmit={(e) => void handleProfileSubmit(e)}
              className="flex flex-col"
            >
              <CardHeader className="px-5 pt-5 pb-4">
                <CardTitle className="text-base font-semibold text-foreground">
                  {t.profile.editProfileHeading}
                </CardTitle>
                <CardDescription className="text-xs leading-relaxed text-muted-foreground">
                  {t.profile.editProfileDescription}
                </CardDescription>
              </CardHeader>

              <CardContent className="space-y-5 px-5 pb-5">
                {profileError ? (
                  <Alert variant="error">{profileError}</Alert>
                ) : null}

                <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-12">
                  <div className="space-y-1.5 sm:col-span-7">
                    <Label
                      htmlFor="profile-display-name"
                      className="text-xs font-medium text-foreground"
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
                      onChange={(e) =>
                        updateField("displayName", e.target.value)
                      }
                      placeholder={t.onboarding.displayNamePlaceholder}
                      className="h-10 rounded-2xl bg-card px-3 text-sm shadow-none"
                    />
                  </div>

                  <div className="space-y-1.5 sm:col-span-5">
                    <Label
                      htmlFor="profile-gender"
                      className="text-xs font-medium text-foreground"
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
                        className="h-10 w-full rounded-2xl border-border bg-card px-3 text-sm shadow-none data-[size=default]:h-10"
                      >
                        <SelectValue>{genderLabels[form.gender]}</SelectValue>
                      </SelectTrigger>
                      <SelectContent
                        alignItemWithTrigger={false}
                        sideOffset={6}
                        className="rounded-[12px] border border-border bg-card p-1 shadow-none ring-0"
                      >
                        <SelectItem
                          value="male"
                          className="rounded-[8px] py-2 pl-2.5 pr-8"
                        >
                          {t.onboarding.genderMale}
                        </SelectItem>
                        <SelectItem
                          value="female"
                          className="rounded-[8px] py-2 pl-2.5 pr-8"
                        >
                          {t.onboarding.genderFemale}
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="grid grid-cols-3 gap-3">
                  <UnitNumberField
                    id="profile-height"
                    label={t.profile.heightShortLabel}
                    unit={t.profile.cmUnit}
                    value={form.heightCm}
                    disabled={profileSaving}
                    onChange={(val) => updateField("heightCm", val)}
                  />
                  <UnitNumberField
                    id="profile-weight"
                    label={t.profile.weightShortLabel}
                    unit={t.profile.kgUnit}
                    value={form.weightKg}
                    disabled={profileSaving}
                    onChange={(val) => updateField("weightKg", val)}
                  />
                  <UnitNumberField
                    id="profile-age"
                    label={t.profile.ageShortLabel}
                    unit={t.profile.ageUnit}
                    value={form.age}
                    integer
                    disabled={profileSaving}
                    onChange={(val) => updateField("age", val)}
                  />
                </div>

                <div className="space-y-2">
                  <Label className="text-xs font-medium text-foreground">
                    {t.onboarding.activityLabel}
                  </Label>
                  <RadioGroup
                    value={form.activityLevel}
                    disabled={profileSaving}
                    onValueChange={(val) =>
                      updateField("activityLevel", val as ActivityLevel)
                    }
                    className="grid grid-cols-3 gap-1.5 sm:gap-2"
                  >
                    {activityOptions.map((opt) => {
                      const selected = form.activityLevel === opt.value;
                      return (
                        <Label
                          key={opt.value}
                          htmlFor={`profile-activity-${opt.value}`}
                          className={`relative flex min-h-10 cursor-pointer flex-col items-start justify-center gap-0.5 rounded-[12px] border px-2.5 py-2 transition-colors duration-150 has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-[var(--brand-primary)] sm:justify-start sm:px-3 sm:py-2 ${
                            selected
                              ? "border-[var(--brand-primary)]/40 bg-[var(--brand-tint)] text-[var(--brand-primary)]"
                              : "border-border bg-card text-foreground hover:bg-accent"
                          }`}
                        >
                          <RadioGroupItem
                            id={`profile-activity-${opt.value}`}
                            value={opt.value}
                            className="sr-only absolute inset-0 size-0 border-0 p-0 after:hidden"
                          />
                          <div className="flex w-full items-center justify-between gap-1.5">
                            <span className="truncate text-xs font-semibold">
                              {opt.label}
                            </span>
                            <ActivityLevelBars level={opt.level} />
                          </div>
                          <span
                            className={`hidden text-[11px] font-normal leading-tight sm:block ${
                              selected
                                ? "text-[var(--brand-primary)]/85"
                                : "text-muted-foreground"
                            }`}
                          >
                            {opt.hint}
                          </span>
                        </Label>
                      );
                    })}
                  </RadioGroup>
                  {(() => {
                    const activeHint = activityOptions.find(
                      (opt) => opt.value === form.activityLevel
                    )?.hint;
                    return activeHint ? (
                      <p className="px-0.5 text-[11px] leading-snug text-muted-foreground sm:hidden">
                        {activeHint}
                      </p>
                    ) : null;
                  })()}
                </div>

                <Separator className="bg-border" />

                <div className="space-y-3">
                  <div className="space-y-1">
                    <h3 className="text-sm font-semibold text-foreground">
                      {t.profile.macroTargetsHeading}
                    </h3>
                    <p className="text-xs leading-relaxed text-muted-foreground">
                      {t.profile.recalcNote}
                    </p>
                  </div>

                  <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
                    <UnitNumberField
                      id="profile-calories"
                      label={t.profile.dailyCalories}
                      unit={t.profile.kcalUnit}
                      value={form.dailyCalories}
                      integer
                      disabled={profileSaving}
                      onChange={(val) => updateField("dailyCalories", val)}
                    />
                    <UnitNumberField
                      id="profile-protein"
                      label={t.profile.protein}
                      unit={t.profile.gramUnit}
                      value={form.proteinG}
                      disabled={profileSaving}
                      onChange={(val) => updateField("proteinG", val)}
                    />
                    <UnitNumberField
                      id="profile-carbs"
                      label={t.profile.carbs}
                      unit={t.profile.gramUnit}
                      value={form.carbsG}
                      disabled={profileSaving}
                      onChange={(val) => updateField("carbsG", val)}
                    />
                    <UnitNumberField
                      id="profile-fat"
                      label={t.profile.fat}
                      unit={t.profile.gramUnit}
                      value={form.fatG}
                      disabled={profileSaving}
                      onChange={(val) => updateField("fatG", val)}
                    />
                    <UnitNumberField
                      id="profile-fiber"
                      label={t.profile.fiber}
                      unit={t.profile.gramUnit}
                      value={form.fiberG}
                      disabled={profileSaving}
                      onChange={(val) => updateField("fiberG", val)}
                    />
                  </div>
                </div>
              </CardContent>

              <CardFooter className="flex flex-wrap items-center justify-between gap-3 rounded-b-3xl border-t border-border bg-muted/40 px-5 py-3.5">
                <p className="text-xs text-muted-foreground">
                  {t.profile.saveProfileHint}
                </p>
                <div className="flex items-center gap-2">
                  {dirty ? (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      disabled={profileSaving}
                      onClick={handleResetForm}
                      className="h-9 rounded-full px-3.5 text-xs shadow-none"
                    >
                      <RotateCcw
                        className="size-3.5 stroke-[1.75]"
                        aria-hidden="true"
                      />
                      <span>{t.profile.resetChangesButton}</span>
                    </Button>
                  ) : null}
                  <Button
                    type="submit"
                    disabled={profileSaving}
                    className="h-9 rounded-full px-5 text-xs font-medium shadow-none"
                  >
                    {profileSaving
                      ? t.common.saving
                      : t.profile.saveProfileButton}
                  </Button>
                </div>
              </CardFooter>
            </form>
          </Card>
        </TabsContent>

        <TabsContent value="security">
          <Card className="rounded-3xl border border-border bg-card py-0 shadow-none ring-0 gap-0">
            {hasPassword ? (
              <form
                noValidate
                onSubmit={(e) => void handlePasswordSubmit(e)}
                className="flex flex-col"
              >
                <CardHeader className="px-5 pt-4 pb-3">
                  <CardTitle className="text-base font-semibold text-foreground">
                    {t.profile.changePasswordHeading}
                  </CardTitle>
                  <CardDescription className="text-xs leading-relaxed text-muted-foreground">
                    {t.profile.changePasswordDescription}
                  </CardDescription>
                </CardHeader>

                <CardContent className="space-y-3.5 px-5 pb-4">
                  {passwordError ? (
                    <Alert variant="error">{passwordError}</Alert>
                  ) : null}

                  <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2">
                    <div className="sm:col-span-2">
                      <PasswordInput
                        id="profile-current-password"
                        name="currentPassword"
                        label={t.common.currentPasswordLabel}
                        value={currentPassword}
                        onChange={(val) => {
                          setPasswordError(null);
                          setCurrentPassword(val);
                        }}
                        autoComplete="current-password"
                        disabled={passwordSaving}
                      />
                    </div>

                    <PasswordInput
                      id="profile-new-password"
                      name="newPassword"
                      label={t.common.newPasswordLabel}
                      value={newPassword}
                      onChange={(val) => {
                        setPasswordError(null);
                        setNewPassword(val);
                      }}
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
                      onChange={(val) => {
                        setPasswordError(null);
                        setConfirmNewPassword(val);
                      }}
                      autoComplete="new-password"
                      minLength={8}
                      disabled={passwordSaving}
                    />
                  </div>
                </CardContent>

                <CardFooter className="flex flex-wrap items-center justify-end gap-3 rounded-b-3xl border-t border-border bg-muted/40 px-5 py-3">
                  <Button
                    type="submit"
                    disabled={passwordSaving}
                    className="h-9 rounded-full px-5 text-xs font-medium shadow-none"
                  >
                    {passwordSaving
                      ? t.common.saving
                      : t.profile.changePasswordButton}
                  </Button>
                </CardFooter>
              </form>
            ) : (
              <>
                <CardHeader className="px-5 pt-5 pb-2">
                  <CardTitle className="text-base font-semibold text-foreground">
                    {t.profile.googleManagedHeading}
                  </CardTitle>
                </CardHeader>
                <CardContent className="px-5 pb-5">
                  <p className="text-sm leading-relaxed text-muted-foreground">
                    {t.profile.googleManagedDescription}
                  </p>
                </CardContent>
              </>
            )}
          </Card>
        </TabsContent>

        <TabsContent value="account">
          <Card className="rounded-3xl border border-border bg-card py-5 shadow-none ring-0 gap-5">
            <CardContent className="space-y-5 px-5">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="max-w-lg space-y-1">
                  <h2 className="text-base font-semibold text-foreground">
                    {t.profile.privacyHeading}
                  </h2>
                  <p className="text-xs leading-relaxed text-muted-foreground">
                    {t.profile.privacyDescription}
                  </p>
                </div>

                <Button
                  render={<Link href="/privacy" />}
                  variant="outline"
                  size="sm"
                  className="h-9 shrink-0 self-start rounded-full border-border bg-card px-4 text-xs font-medium text-foreground shadow-none hover:bg-accent sm:self-center"
                >
                  <span>{t.profile.privacyLinkText}</span>
                  <ArrowUpRight
                    className="size-3.5 stroke-[1.75]"
                    aria-hidden="true"
                  />
                </Button>
              </div>

              <Separator className="bg-border" />

              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="max-w-lg space-y-1">
                  <h2 className="text-base font-semibold text-foreground">
                    {t.profile.deleteAccountHeading}
                  </h2>
                  <p className="text-xs leading-relaxed text-muted-foreground">
                    {hasPassword
                      ? t.profile.deleteAccountDescription
                      : t.profile.deleteAccountGoogleDescription}
                  </p>
                </div>

                <Button
                  type="button"
                  variant="destructive"
                  size="sm"
                  onClick={() => {
                    setDeletePassword("");
                    setDeleteError(null);
                    setDeleteModalOpen(true);
                  }}
                  className="h-9 shrink-0 self-start rounded-full bg-[var(--error-bg)] px-4 text-xs font-medium text-[var(--error-text)] shadow-none hover:bg-[var(--error-bg)]/80 sm:self-center"
                >
                  <Trash2
                    className="size-3.5 stroke-[1.75]"
                    aria-hidden="true"
                  />
                  <span>{t.profile.deleteAccountButton}</span>
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

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

          {hasPassword ? (
            <form
              noValidate
              onSubmit={(e) => void handleDeleteAccount(e)}
              className="space-y-4"
            >
              <PasswordInput
                id="delete-account-password"
                name="deletePassword"
                label={t.profile.deleteDialogPasswordLabel}
                value={deletePassword}
                onChange={(val) => {
                  setDeleteError(null);
                  setDeletePassword(val);
                }}
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
                  className="h-10 rounded-full bg-[var(--error-bg)] px-5 text-sm font-medium text-[var(--error-text)] shadow-none hover:bg-[var(--error-bg)]/80"
                >
                  {deleteSubmitting
                    ? t.profile.deleteDialogDeleting
                    : t.profile.deleteDialogConfirm}
                </Button>
              </AlertDialogFooter>
            </form>
          ) : (
            <div className="space-y-4">
              <p className="text-xs leading-relaxed text-muted-foreground">
                {t.profile.deleteDialogGoogleDescription}
              </p>

              {deleteModalOpen ? (
                <SocialAuthButtons
                  mode="reauth"
                  showDivider={false}
                  disabled={deleteSubmitting}
                  onCredential={handleGoogleDeleteAccount}
                  onError={(code) =>
                    setDeleteError(getFriendlyErrorMessage(code, language))
                  }
                />
              ) : null}

              {deleteSubmitting ? (
                <p className="text-center text-xs text-muted-foreground">
                  {t.profile.deleteDialogDeleting}
                </p>
              ) : null}

              <AlertDialogFooter className="mx-0 mb-0 rounded-none border-0 bg-transparent p-0 pt-2">
                <AlertDialogCancel
                  disabled={deleteSubmitting}
                  className="h-10 rounded-full px-5 shadow-none"
                >
                  {t.common.cancel}
                </AlertDialogCancel>
              </AlertDialogFooter>
            </div>
          )}
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
