"use client";

import type { ReactNode } from "react";
import { AuthProvider } from "@/components/providers/AuthProvider";
import { I18nProvider } from "@/components/providers/I18nProvider";
import { ThemeProvider, useTheme } from "@/components/providers/ThemeProvider";
import { Toaster } from "@/components/ui/sonner";

function ThemedToaster() {
  const { theme } = useTheme();
  return (
    <Toaster
      theme={theme}
      position="bottom-right"
      toastOptions={{
        classNames: {
          toast:
            "rounded-2xl shadow-none border border-border bg-card text-card-foreground",
        },
      }}
    />
  );
}

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <AuthProvider>
      <I18nProvider>
        <ThemeProvider>
          {children}
          <ThemedToaster />
        </ThemeProvider>
      </I18nProvider>
    </AuthProvider>
  );
}
