import type { ReactNode } from "react";
import { OrganicWaveBackdrop } from "@/components/auth/AuthIllustrations";
import { AuthHeader } from "@/components/layout/AuthHeader";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="relative flex min-h-dvh flex-col bg-background text-foreground overflow-x-hidden selection:bg-[var(--brand-tint)] selection:text-[var(--brand-primary)]">
      <OrganicWaveBackdrop />
      <AuthHeader />
      <main className="relative z-10 mx-auto flex w-full max-w-6xl flex-1 flex-col justify-center px-4 py-8 sm:px-6 lg:px-8">
        {children}
      </main>
    </div>
  );
}
