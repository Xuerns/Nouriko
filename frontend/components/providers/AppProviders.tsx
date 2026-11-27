"use client";

import { useSyncExternalStore, type ReactNode } from "react";
import { usePathname } from "next/navigation";
import { AuthProvider } from "@/components/providers/AuthProvider";
import { I18nProvider } from "@/components/providers/I18nProvider";
import { ThemeProvider, useTheme } from "@/components/providers/ThemeProvider";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";

const MOBILE_NAV_QUERY = "(max-width: 767px)";
const APP_SHELL_PATHS = new Set(["/chat", "/summary", "/history", "/profile"]);

function subscribeMobileNav(onStoreChange: () => void) {
  const mediaQuery = window.matchMedia(MOBILE_NAV_QUERY);
  mediaQuery.addEventListener("change", onStoreChange);
  return () => mediaQuery.removeEventListener("change", onStoreChange);
}

function getMobileNavSnapshot() {
  return window.matchMedia(MOBILE_NAV_QUERY).matches;
}

function getMobileNavServerSnapshot() {
  return false;
}

function ThemedToaster() {
  const { theme } = useTheme();
  const pathname = usePathname();
  const isMobile = useSyncExternalStore(
    subscribeMobileNav,
    getMobileNavSnapshot,
    getMobileNavServerSnapshot
  );

  const mobileTopOffset = APP_SHELL_PATHS.has(pathname) ? "4.75rem" : "1rem";
  const mobileToastOffset = {
    top: mobileTopOffset,
    left: "1rem",
    right: "1rem",
  };

  return (
    <Toaster
      theme={theme}
      position={isMobile ? "top-center" : "bottom-right"}
      offset={isMobile ? mobileToastOffset : undefined}
      mobileOffset={mobileToastOffset}
      toastOptions={{
        style: { boxShadow: "none" },
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
          <TooltipProvider delay={250}>
            {children}
            <ThemedToaster />
          </TooltipProvider>
        </ThemeProvider>
      </I18nProvider>
    </AuthProvider>
  );
}
