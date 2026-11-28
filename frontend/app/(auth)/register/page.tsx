"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft, ArrowRight, Mail } from "lucide-react";
import {
  RegisterSaladIllustration,
  SuccessCheckIllustration,
} from "@/components/auth/AuthIllustrations";
import { SocialAuthButtons } from "@/components/auth/SocialAuthButtons";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";
import { Alert } from "@/components/ui/Alert";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { FieldError } from "@/components/ui/FieldError";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AuthSplitSkeleton } from "@/components/ui/PageSkeletons";
import { PasswordInput } from "@/components/ui/PasswordInput";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";
import { focusFieldById, isValidEmailFormat } from "@/lib/form-validation";
import { cn } from "@/lib/utils";

export default function RegisterPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading, loginWithGoogle, profile } = useAuth();
  const { language, t } = useI18n();
  const { theme } = useTheme();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [emailError, setEmailError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [confirmError, setConfirmError] = useState<string | null>(null);
  const [dirty, setDirty] = useState({
    email: false,
    password: false,
    confirm: false,
  });
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [registered, setRegistered] = useState(false);
  const [devToken, setDevToken] = useState<string | null>(null);

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.replace(profile?.onboarding_completed ? "/chat" : "/onboarding");
    }
  }, [isAuthenticated, isLoading, profile, router]);

  function validateEmail(value: string): string | null {
    const trimmed = value.trim();
    if (!trimmed) return t.common.emailRequired;
    if (!isValidEmailFormat(trimmed)) return t.common.emailInvalid;
    return null;
  }

  function validatePassword(value: string): string | null {
    if (!value) return t.common.passwordRequired;
    if (value.length < 8) return t.common.passwordMinHint;
    return null;
  }

  function validateConfirm(value: string, currentPassword: string): string | null {
    if (!value) return t.common.confirmPasswordRequired;
    if (value !== currentPassword) return t.common.passwordMismatch;
    return null;
  }

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);

    const nextEmailErr = validateEmail(email);
    const nextPassErr = validatePassword(password);
    const nextConfirmErr = validateConfirm(confirmPassword, password);

    setEmailError(nextEmailErr);
    setPasswordError(nextPassErr);
    setConfirmError(nextConfirmErr);

    if (nextEmailErr) {
      focusFieldById("register-email");
      return;
    }
    if (nextPassErr) {
      focusFieldById("register-password");
      return;
    }
    if (nextConfirmErr) {
      focusFieldById("register-confirm-password");
      return;
    }

    setSubmitting(true);
    try {
      const res = await apiClient.register({
        email: email.trim(),
        password,
        language,
        theme,
      });
      setRegistered(true);
      setDevToken(res.dev_verification_token ?? null);
    } catch (err) {
      const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
      setError(getFriendlyErrorMessage(code, language));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleGoogleCredential({
    idToken,
    nonce,
  }: {
    idToken: string;
    nonce: string;
  }) {
    setError(null);
    setSubmitting(true);

    try {
      const loggedInProfile = await loginWithGoogle({
        id_token: idToken,
        nonce,
        language,
        theme,
      });
      router.replace(
        loggedInProfile.onboarding_completed ? "/chat" : "/onboarding"
      );
    } catch (err) {
      const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
      setError(getFriendlyErrorMessage(code, language));
    } finally {
      setSubmitting(false);
    }
  }

  if (isLoading || isAuthenticated) {
    return <AuthSplitSkeleton label={t.common.loading} />;
  }

  if (registered) {
    return (
      <div className="mx-auto w-full max-w-md py-6">
        <Card className="w-full rounded-3xl border border-border/80 bg-card p-7 sm:p-10 shadow-none text-center space-y-6">
          <SuccessCheckIllustration />

          <div className="space-y-2">
            <h2 className="text-2xl font-bold tracking-tight text-foreground">
              {t.register.successTitle}
            </h2>
            <p className="text-sm text-muted-foreground leading-relaxed">
              {t.register.successMessage}
            </p>
          </div>

          {devToken ? (
            <div className="rounded-2xl border border-border bg-muted/40 p-4 space-y-3 text-sm text-left">
              <p className="text-xs font-medium text-muted-foreground">
                {t.register.devTokenLabel}
              </p>
              <div>
                <Link
                  href={`/verify-email?token=${encodeURIComponent(devToken)}`}
                  className={cn(
                    buttonVariants({ size: "sm" }),
                    "rounded-full shadow-none bg-[var(--brand-primary)] hover:bg-[var(--brand-primary-hover)] text-white"
                  )}
                >
                  {t.register.devVerifyLink}
                </Link>
              </div>
            </div>
          ) : (
            <div>
              <Link
                href="/verify-email"
                className={cn(
                  buttonVariants({ variant: "outline" }),
                  "h-11 rounded-full shadow-none border-border"
                )}
              >
                {t.register.goToVerify}
              </Link>
            </div>
          )}

          <div className="border-t border-border pt-4 text-center text-sm text-muted-foreground">
            <p>
              {t.register.hasAccount}{" "}
              <Link
                href="/login"
                className="font-semibold text-foreground underline underline-offset-4 hover:text-[var(--brand-primary)] transition-colors"
              >
                {t.register.loginLink}
              </Link>
            </p>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-3 py-1 lg:space-y-4 lg:py-4">
      <div className="mx-auto w-full max-w-md lg:max-w-none">
        <Button
          render={<Link href="/" />}
          nativeButton={false}
          variant="ghost"
          size="sm"
          className="h-9 rounded-full px-3 text-xs font-medium text-muted-foreground shadow-none hover:bg-accent hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" strokeWidth={1.75} aria-hidden="true" />
          <span>{t.common.backHome}</span>
        </Button>
      </div>

      <div className="grid grid-cols-1 items-center gap-8 lg:grid-cols-12 lg:gap-14">
        {/* Left Column: Inspiring Hero & Salad Illustration (desktop only so form is immediately visible on mobile) */}
        <div className="hidden flex-col items-start text-left lg:col-span-7 lg:flex space-y-6">
          <div className="space-y-3">
            <h1 className="text-3xl font-extrabold tracking-tight text-foreground sm:text-4xl lg:text-5xl leading-[1.15]">
              {t.register.heroTitle}{" "}
              <span className="text-[var(--brand-primary)]">
                {t.register.heroBrand}
              </span>{" "}
              <span className="text-[var(--brand-accent)]">
                {t.register.heroAccent}
              </span>
            </h1>
            <p className="max-w-md text-sm text-muted-foreground sm:text-base leading-relaxed">
              {t.register.heroSubtitle}
            </p>
          </div>

          {/* Salad Illustration */}
          <div className="pt-2 w-full flex justify-center lg:justify-start">
            <RegisterSaladIllustration />
          </div>
        </div>

        {/* Right Column: Form Card */}
        <div className="mx-auto w-full max-w-md lg:col-span-5 lg:max-w-none">
          <Card className="w-full rounded-3xl border border-border/80 bg-card p-5 sm:p-8 lg:p-9 shadow-none space-y-5 sm:space-y-6">
            <div className="space-y-1">
              <h2 className="text-2xl font-bold tracking-tight text-foreground">
                {t.register.title}
              </h2>
              <p className="text-sm text-muted-foreground">
                {t.register.subtitle}
              </p>
            </div>

            {error ? <Alert variant="error">{error}</Alert> : null}

            <form
              noValidate
              onSubmit={(e) => void handleSubmit(e)}
              className="space-y-4"
            >
              {/* Email with Mail prefix icon */}
              <div className="space-y-1.5">
                <Label
                  htmlFor="register-email"
                  className="text-sm font-medium text-foreground"
                >
                  {t.common.emailLabel}
                </Label>
                <div className="relative">
                  <Mail
                    className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground"
                    strokeWidth={1.75}
                    aria-hidden="true"
                  />
                  <Input
                    id="register-email"
                    name="email"
                    type="email"
                    autoComplete="email"
                    autoCorrect="off"
                    autoCapitalize="none"
                    spellCheck={false}
                    required
                    disabled={submitting}
                    value={email}
                    onChange={(e) => {
                      const next = e.target.value;
                      setEmail(next);
                      setDirty((prev) => ({ ...prev, email: true }));
                      if (emailError && !validateEmail(next)) {
                        setEmailError(null);
                      }
                    }}
                    onBlur={() => {
                      if (dirty.email) {
                        setEmailError(validateEmail(email));
                      }
                    }}
                    aria-invalid={Boolean(emailError)}
                    aria-describedby={
                      emailError ? "register-email-error" : undefined
                    }
                    placeholder={t.common.emailPlaceholder}
                    className={cn(
                      "h-12 w-full rounded-2xl border-border bg-card pl-10 pr-4 text-sm shadow-none focus-visible:border-[var(--brand-primary)] focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)]/20",
                      emailError &&
                        "border-[var(--error-text)] focus-visible:border-[var(--error-text)]"
                    )}
                  />
                </div>
                <FieldError id="register-email-error" message={emailError} />
              </div>

              {/* Password */}
              <PasswordInput
                id="register-password"
                name="password"
                label={t.common.passwordLabel}
                value={password}
                onChange={(next) => {
                  setPassword(next);
                  setDirty((prev) => ({ ...prev, password: true }));
                  if (passwordError && !validatePassword(next)) {
                    setPasswordError(null);
                  }
                  if (confirmError && !validateConfirm(confirmPassword, next)) {
                    setConfirmError(null);
                  }
                }}
                onBlur={() => {
                  if (dirty.password) {
                    setPasswordError(validatePassword(password));
                  }
                }}
                autoComplete="new-password"
                minLength={8}
                disabled={submitting}
                showStrength
                hint={t.common.passwordMinHint}
                error={passwordError}
                placeholder={t.common.passwordLabel}
              />

              {/* Confirm Password */}
              <PasswordInput
                id="register-confirm-password"
                name="confirmPassword"
                label={t.common.confirmPasswordLabel}
                value={confirmPassword}
                onChange={(next) => {
                  setConfirmPassword(next);
                  setDirty((prev) => ({ ...prev, confirm: true }));
                  if (confirmError && !validateConfirm(next, password)) {
                    setConfirmError(null);
                  }
                }}
                onBlur={() => {
                  if (dirty.confirm) {
                    setConfirmError(validateConfirm(confirmPassword, password));
                  }
                }}
                autoComplete="new-password"
                minLength={8}
                disabled={submitting}
                error={confirmError}
                placeholder={t.common.confirmPasswordLabel}
              />

              {/* Submit Button */}
              <Button
                type="submit"
                disabled={submitting}
                className="h-12 w-full rounded-full font-medium text-base shadow-none transition-all duration-150 bg-[var(--brand-primary)] hover:bg-[var(--brand-primary-hover)] text-white flex items-center justify-center gap-2 cursor-pointer mt-2"
              >
                <span>
                  {submitting ? t.common.submitting : t.register.submitButton}
                </span>
                <ArrowRight className="h-4 w-4" strokeWidth={2} />
              </Button>
            </form>

            <SocialAuthButtons
              mode="register"
              disabled={submitting}
              onCredential={handleGoogleCredential}
              onError={(code) =>
                setError(getFriendlyErrorMessage(code, language))
              }
            />

            {/* Footer with Privacy and Login Link */}
            <div className="space-y-2 border-t border-border pt-4 text-center text-xs text-muted-foreground">
              <p>
                {t.register.privacyPrefix}{" "}
                <Link
                  href="/terms"
                  className="text-foreground underline underline-offset-4 hover:text-[var(--brand-primary)] transition-colors"
                >
                  {t.common.termsOfService}
                </Link>{" "}
                {t.register.legalConnector}{" "}
                <Link
                  href="/privacy"
                  className="text-foreground underline underline-offset-4 hover:text-[var(--brand-primary)] transition-colors"
                >
                  {t.common.privacyPolicy}
                </Link>
                .
              </p>
              <p className="text-sm">
                {t.register.hasAccount}{" "}
                <Link
                  href="/login"
                  className="font-semibold text-foreground underline underline-offset-4 hover:text-[var(--brand-primary)] transition-colors"
                >
                  {t.register.loginLink}
                </Link>
              </p>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}
