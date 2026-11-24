"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { ArrowLeft, Mail } from "lucide-react";
import {
  ForgotPaperPlaneIllustration,
  SuccessCheckIllustration,
} from "@/components/auth/AuthIllustrations";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { apiClient, ApiError } from "@/lib/api";
import { getFriendlyErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";

export default function ForgotPasswordPage() {
  const { language, t } = useI18n();
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [devResetToken, setDevResetToken] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      const res = await apiClient.forgotPassword(email.trim());
      setSubmitted(true);
      setDevResetToken(res.dev_reset_token ?? null);
    } catch (err) {
      const code = err instanceof ApiError ? err.code : "INTERNAL_ERROR";
      setError(getFriendlyErrorMessage(code, language));
    } finally {
      setSubmitting(false);
    }
  }

  if (submitted) {
    return (
      <div className="mx-auto w-full max-w-md py-6">
        <Card className="w-full rounded-3xl border border-border/80 bg-card p-7 sm:p-10 shadow-none text-center space-y-6">
          <SuccessCheckIllustration />

          <div className="space-y-2">
            <h2 className="text-2xl font-bold tracking-tight text-foreground">
              {t.forgotPassword.successTitle}
            </h2>
            <p className="text-sm text-muted-foreground leading-relaxed">
              {t.forgotPassword.successSubtitle}
            </p>
          </div>

          {devResetToken ? (
            <div className="rounded-2xl border border-border bg-muted/40 p-4 space-y-3 text-sm text-left">
              <p className="text-xs font-medium text-muted-foreground">
                Tautan reset lokal (mode pengembang):
              </p>
              <div>
                <Link
                  href={`/reset-password?token=${encodeURIComponent(devResetToken)}`}
                  className={cn(
                    buttonVariants({ size: "sm" }),
                    "rounded-full shadow-none bg-[var(--brand-primary)] hover:bg-[var(--brand-primary-hover)] text-white"
                  )}
                >
                  {t.forgotPassword.devResetLink}
                </Link>
              </div>
            </div>
          ) : null}

          <div className="pt-2">
            <Link
              href="/login"
              className={cn(
                buttonVariants(),
                "h-12 w-full rounded-full font-medium text-base shadow-none bg-[var(--brand-primary)] hover:bg-[var(--brand-primary-hover)] text-white flex items-center justify-center gap-2 cursor-pointer"
              )}
            >
              {t.forgotPassword.backToLoginBtn}
            </Link>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 items-center gap-10 lg:grid-cols-12 lg:gap-14 py-4">
      {/* Left Column: Envelope Illustration & Headline */}
      <div className="flex flex-col items-center text-center lg:col-span-7 lg:items-start lg:text-left space-y-6">
        <div className="space-y-3">
          <h1 className="text-3xl font-extrabold tracking-tight text-foreground sm:text-4xl lg:text-5xl leading-[1.15]">
            {t.forgotPassword.heroTitle}
          </h1>
          <p className="max-w-md text-sm text-muted-foreground sm:text-base leading-relaxed">
            {t.forgotPassword.heroSubtitle}
          </p>
        </div>

        {/* Paper Plane Illustration */}
        <div className="pt-2 w-full flex justify-center lg:justify-start">
          <ForgotPaperPlaneIllustration />
        </div>
      </div>

      {/* Right Column: Form Card */}
      <div className="w-full lg:col-span-5">
        <Card className="w-full rounded-3xl border border-border/80 bg-card p-6 sm:p-9 shadow-none space-y-6">
          <div className="space-y-1">
            <h2 className="text-2xl font-bold tracking-tight text-foreground">
              {t.forgotPassword.title}
            </h2>
            <p className="text-sm text-muted-foreground">
              {t.forgotPassword.subtitle}
            </p>
          </div>

          {error ? <Alert variant="error">{error}</Alert> : null}

          <form onSubmit={(e) => void handleSubmit(e)} className="space-y-5">
            <div className="space-y-1.5">
              <Label
                htmlFor="forgot-email"
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
                  id="forgot-email"
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

            <Button
              type="submit"
              disabled={submitting}
              className="h-12 w-full rounded-full font-medium text-base shadow-none transition-all duration-150 bg-[var(--brand-primary)] hover:bg-[var(--brand-primary-hover)] text-white flex items-center justify-center gap-2 cursor-pointer mt-2"
            >
              <span>{submitting ? t.common.submitting : t.forgotPassword.submitButton}</span>
            </Button>
          </form>

          {/* Back to Login */}
          <div className="border-t border-border pt-4 text-center text-sm">
            <Link
              href="/login"
              className="inline-flex items-center gap-1.5 font-medium text-muted-foreground hover:text-foreground transition-colors"
            >
              <ArrowLeft className="h-4 w-4" strokeWidth={1.75} />
              <span>{t.forgotPassword.backToLogin}</span>
            </Link>
          </div>
        </Card>
      </div>
    </Card>
  );
}
