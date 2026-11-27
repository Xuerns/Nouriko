"use client";

import { Suspense, useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { ArrowLeft, ArrowRight, Check, Mail } from "lucide-react";
import { LoginMealIllustration } from "@/components/auth/AuthIllustrations";
import { SocialAuthButtons } from "@/components/auth/SocialAuthButtons";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { useTheme } from "@/components/providers/ThemeProvider";
import { Alert } from "@/components/ui/Alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { FieldError } from "@/components/ui/FieldError";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AuthSplitSkeleton } from "@/components/ui/PageSkeletons";
import { PasswordInput } from "@/components/ui/PasswordInput";
import { ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";
import { focusFieldById, isValidEmailFormat } from "@/lib/form-validation";
import { cn } from "@/lib/utils";

function LoginContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { isAuthenticated, isLoading, login, loginWithGoogle, profile } =
    useAuth();
  const { language, t } = useI18n();
  const { theme } = useTheme();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [dirty, setDirty] = useState({ email: false, password: false });
  const [rememberMe, setRememberMe] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const sessionExpired = searchParams.get("reason") === "session_expired";

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
    return value ? null : t.common.passwordRequired;
  }

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);

    const nextEmailErr = validateEmail(email);
    const nextPassErr = validatePassword(password);
    setEmailError(nextEmailErr);
    setPasswordError(nextPassErr);

    if (nextEmailErr) {
      focusFieldById("login-email");
      return;
    }
    if (nextPassErr) {
      focusFieldById("login-password");
      return;
    }

    setSubmitting(true);

    try {
      const loggedInProfile = await login(email.trim(), password);
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

  return (
    <div className="space-y-4 py-4">
      <div>
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

      <div className="grid grid-cols-1 items-center gap-10 lg:grid-cols-12 lg:gap-14">
        {/* Left Column: Inspiring Hero & Meal Illustration */}
        <div className="flex flex-col items-center text-center lg:col-span-7 lg:items-start lg:text-left space-y-6">
          <div className="space-y-3">
            <h1 className="text-3xl font-extrabold tracking-tight text-foreground sm:text-4xl lg:text-5xl leading-[1.15]">
              {t.login.heroTitle}{" "}
              <span className="text-[var(--brand-accent)]">
                {t.login.heroAccent}
              </span>
            </h1>
            <p className="max-w-md text-sm text-muted-foreground sm:text-base leading-relaxed">
              {t.login.heroSubtitle}
            </p>
          </div>

          {/* Feature Checkmarks */}
          <div className="flex flex-col space-y-2.5 pt-1 text-sm font-medium text-foreground">
            <div className="flex items-center gap-2.5">
              <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--brand-tint)] text-[var(--brand-primary)]">
                <Check className="h-3.5 w-3.5" strokeWidth={2.5} />
              </span>
              <span>{t.login.feature1}</span>
            </div>
            <div className="flex items-center gap-2.5">
              <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--brand-tint)] text-[var(--brand-primary)]">
                <Check className="h-3.5 w-3.5" strokeWidth={2.5} />
              </span>
              <span>{t.login.feature2}</span>
            </div>
            <div className="flex items-center gap-2.5">
              <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--brand-tint)] text-[var(--brand-primary)]">
                <Check className="h-3.5 w-3.5" strokeWidth={2.5} />
              </span>
              <span>{t.login.feature3}</span>
            </div>
          </div>

          {/* Meal Illustration */}
          <div className="pt-2 w-full flex justify-center lg:justify-start">
            <LoginMealIllustration />
          </div>
        </div>

        {/* Right Column: Form Card */}
        <div className="w-full lg:col-span-5">
          <Card className="w-full rounded-3xl border border-border/80 bg-card p-6 sm:p-9 shadow-none space-y-6">
            <div className="space-y-1">
              <h2 className="text-2xl font-bold tracking-tight text-foreground">
                {t.login.title}
              </h2>
              <p className="text-sm text-muted-foreground">
                {t.login.subtitle}
              </p>
            </div>

            {sessionExpired && !error ? (
              <Alert variant="info">{t.login.sessionExpired}</Alert>
            ) : null}

            {error ? <Alert variant="error">{error}</Alert> : null}

            <form
              noValidate
              onSubmit={(e) => void handleSubmit(e)}
              className="space-y-4"
            >
              {/* Email with Mail prefix icon */}
              <div className="space-y-1.5">
                <Label
                  htmlFor="login-email"
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
                    id="login-email"
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
                    aria-describedby={emailError ? "login-email-error" : undefined}
                    placeholder={t.common.emailPlaceholder}
                    className={cn(
                      "h-12 w-full rounded-2xl border-border bg-card pl-10 pr-4 text-sm shadow-none focus-visible:border-[var(--brand-primary)] focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)]/20",
                      emailError &&
                        "border-[var(--error-text)] focus-visible:border-[var(--error-text)]"
                    )}
                  />
                </div>
                <FieldError id="login-email-error" message={emailError} />
              </div>

              {/* Password */}
              <PasswordInput
                id="login-password"
                name="password"
                label={t.common.passwordLabel}
                value={password}
                onChange={(next) => {
                  setPassword(next);
                  setDirty((prev) => ({ ...prev, password: true }));
                  if (passwordError && !validatePassword(next)) {
                    setPasswordError(null);
                  }
                }}
                onBlur={() => {
                  if (dirty.password) {
                    setPasswordError(validatePassword(password));
                  }
                }}
                autoComplete="current-password"
                disabled={submitting}
                error={passwordError}
                placeholder={t.common.passwordLabel}
              />

              {/* Remember me & Forgot Password */}
              <div className="flex items-center justify-between pt-1">
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="remember-me"
                    checked={rememberMe}
                    onCheckedChange={(checked) => setRememberMe(Boolean(checked))}
                    disabled={submitting}
                  />
                  <Label
                    htmlFor="remember-me"
                    className="cursor-pointer text-xs font-normal text-muted-foreground select-none"
                  >
                    {t.login.rememberMe}
                  </Label>
                </div>
                <Link
                  href="/forgot-password"
                  className="text-xs font-medium text-[var(--brand-primary)] hover:underline underline-offset-4 transition-colors"
                >
                  {t.login.forgotPasswordLink}
                </Link>
              </div>

              {/* Submit Button */}
              <Button
                type="submit"
                disabled={submitting}
                className="h-12 w-full rounded-full font-medium text-base shadow-none transition-all duration-150 bg-[var(--brand-primary)] hover:bg-[var(--brand-primary-hover)] text-white flex items-center justify-center gap-2 cursor-pointer"
              >
                <span>
                  {submitting ? t.common.submitting : t.login.submitButton}
                </span>
                <ArrowRight className="h-4 w-4" strokeWidth={2} />
              </Button>
            </form>

            <SocialAuthButtons
              mode="login"
              disabled={submitting}
              onCredential={handleGoogleCredential}
              onError={(code) =>
                setError(getFriendlyErrorMessage(code, language))
              }
            />

            {/* Footer switcher */}
            <div className="border-t border-border pt-4 text-center text-sm text-muted-foreground">
              <p>
                {t.login.noAccount}{" "}
                <Link
                  href="/register"
                  className="font-semibold text-foreground underline underline-offset-4 hover:text-[var(--brand-primary)] transition-colors"
                >
                  {t.login.registerLink}
                </Link>
              </p>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={<AuthSplitSkeleton />}>
      <LoginContent />
    </Suspense>
  );
}
