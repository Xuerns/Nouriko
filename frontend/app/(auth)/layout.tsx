import type { ReactNode } from "react";
import { AuthHeader } from "@/components/layout/AuthHeader";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col bg-background text-foreground">
      <AuthHeader />
      <main className="mx-auto flex w-full max-w-5xl flex-1 flex-col justify-center px-4 py-8 sm:px-6">
        {children}
      </main>
    </div>
  );
}
