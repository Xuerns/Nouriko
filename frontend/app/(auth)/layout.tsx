import type { ReactNode } from "react";
import { AuthHeader } from "@/components/layout/AuthHeader";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col bg-white text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100">
      <AuthHeader />
      <main className="mx-auto flex w-full max-w-4xl flex-1 flex-col justify-center px-4 py-8">
        {children}
      </main>
    </div>
  );
}
