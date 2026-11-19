"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useAuth } from "@/components/providers/AuthProvider";
import { apiClient } from "@/lib/api";
import type { Theme } from "@/lib/types";

const THEME_STORAGE_KEY = "nouriko_theme";

interface ThemeContextValue {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

function detectInitialTheme(): Theme {
  if (typeof window === "undefined") {
    return "light";
  }
  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY);
    if (stored === "light" || stored === "dark") {
      return stored;
    }
  } catch {
    // Ignore storage errors.
  }
  if (
    window.matchMedia &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
  ) {
    return "dark";
  }
  return "light";
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const { token, profile, setProfile, handleUnauthorized } = useAuth();
  const [localTheme, setLocalTheme] = useState<Theme>("light");
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    const detected = detectInitialTheme();
    const timer = window.setTimeout(() => {
      setLocalTheme(detected);
      setHydrated(true);
    }, 0);
    return () => window.clearTimeout(timer);
  }, []);

  const theme: Theme =
    profile?.theme === "light" || profile?.theme === "dark"
      ? profile.theme
      : localTheme;

  useEffect(() => {
    document.documentElement.classList.toggle("dark", theme === "dark");
    if (hydrated || profile) {
      try {
        localStorage.setItem(THEME_STORAGE_KEY, theme);
      } catch {
        // Ignore storage errors.
      }
    }
  }, [hydrated, profile, theme]);

  const setTheme = useCallback(
    (nextTheme: Theme) => {
      setLocalTheme(nextTheme);
      try {
        localStorage.setItem(THEME_STORAGE_KEY, nextTheme);
      } catch {
        // Ignore storage errors.
      }

      if (token && profile) {
        setProfile({ ...profile, theme: nextTheme });
        void apiClient
          .updateProfile(token, { theme: nextTheme })
          .then((res) => {
            setProfile(res.profile);
          })
          .catch((err) => {
            handleUnauthorized(err);
          });
      }
    },
    [handleUnauthorized, profile, setProfile, token]
  );

  const toggleTheme = useCallback(() => {
    setTheme(theme === "light" ? "dark" : "light");
  }, [setTheme, theme]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      theme,
      setTheme,
      toggleTheme,
    }),
    [setTheme, theme, toggleTheme]
  );

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return ctx;
}
