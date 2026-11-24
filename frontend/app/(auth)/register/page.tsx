"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowRight, Mail } from "lucide-react";
import {
  RegisterSaladIllustration,
  SuccessCheckIllustration,
} from "@/components/auth/AuthIllustrations";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";
import { Alert } from "@/components/ui/Alert";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PasswordInput } from "@/components/ui/PasswordInput";
import { Separator } from "@/components/ui/separator";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";

export default function RegisterPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading, profile } = useAuth();
  const { language, t } = useI18n();
  const { theme } = useTheme();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [registered, setRegistered] = useState(false);
  const [devToken, setDevToken] = useState<string | null>(null);

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.replace(profile?.onboarding_completed ? "/chat" : "/onboarding");
    }
  }, [isAuthenticated, isLoading, profile, router]);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);

    if (password !== confirmPassword) {
      setError(t.common.passwordMismatch);
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

  if (isLoading || isAuthenticated) {
    return (
      <div className="mx-auto w-full max-w-md rounded-3xl border border-border bg-card p-8 text-center text-sm text-muted-foreground shadow-none">
        {t.common.loading}
      </Card>
    );
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
    <div className="grid grid-cols-1 items-center gap-10 lg:grid-cols-12 lg:gap-14 py-4">
      {/* Left Column: Inspiring Hero & Salad Illustration */}
      <div className="flex flex-col items-center text-center lg:col-span-7 lg:items-start lg:text-left space-y-6">
        <div className="space-y-3">
          <h1 className="text-3xl font-extrabold tracking-tight text-foreground sm:text-4xl lg:text-5xl leading-[1.15]">
            {t.register.heroTitle}{" "}
            <span className="text-[var(--brand-primary)]">{t.register.heroBrand}</span>{" "}
            <span className="text-[var(--brand-accent)]">{t.register.heroAccent}</span>
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
      <div className="w-full lg:col-span-5">
        <Card className="w-full rounded-3xl border border-border/80 bg-card p-6 sm:p-9 shadow-none space-y-6">
          <div className="space-y-1">
            <h2 className="text-2xl font-bold tracking-tight text-foreground">
              {t.register.title}
            </h2>
            <p className="text-sm text-muted-foreground">
              {t.register.subtitle}
            </p>
          </div>

          {error ? <Alert variant="error">{error}</Alert> : null}

          <form onSubmit={(e) => void handleSubmit(e)} className="space-y-4">
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
                  required
                  disabled={submitting}
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder={t.common.emailPlaceholder}
                  className="h-12 w-full rounded-2xl border-border bg-card pl-10 pr-4 text-sm shadow-none focus-visible:border-[var(--brand-primary)] focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)]/20"
                />
              </div>
            </div>

            {/* Password */}
            <PasswordInput
              id="register-password"
              name="password"
              label={t.common.passwordLabel}
              value={password}
              onChange={setPassword}
              autoComplete="new-password"
              minLength={8}
              disabled={submitting}
              showStrength
              hint={t.common.passwordMinHint}
              placeholder={t.common.passwordLabel}
            />

            {/* Confirm Password */}
            <PasswordInput
              id="register-confirm-password"
              name="confirmPassword"
              label={t.common.confirmPasswordLabel}
              value={confirmPassword}
              onChange={setConfirmPassword}
              autoComplete="new-password"
              minLength={8}
              disabled={submitting}
              placeholder={t.common.confirmPasswordLabel}
            />

            {/* Submit Button */}
            <Button
              type="submit"
              disabled={submitting}
              className="h-12 w-full rounded-full font-medium text-base shadow-none transition-all duration-150 bg-[var(--brand-primary)] hover:bg-[var(--brand-primary-hover)] text-white flex items-center justify-center gap-2 cursor-pointer mt-2"
            >
              <span>{submitting ? t.common.submitting : t.register.submitButton}</span>
              <ArrowRight className="h-4 w-4" strokeWidth={2} />
            </Button>
          </form>

          {/* Footer with Privacy and Login Link */}
          <div className="space-y-2 border-t border-border pt-4 text-center text-xs text-muted-foreground">
            <p>
              {t.register.privacyPrefix}{" "}
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
    </Card>
  );
}
