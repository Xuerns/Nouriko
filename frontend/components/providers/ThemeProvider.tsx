"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type MouseEvent as ReactMouseEvent,
  type ReactNode,
} from "react";
import { flushSync } from "react-dom";
import { useAuth } from "@/components/providers/AuthProvider";
import { apiClient } from "@/lib/api";
import type { Theme } from "@/lib/types";

const THEME_STORAGE_KEY = "nouriko_theme";

export type ThemeTransitionOrigin =
  | ReactMouseEvent<HTMLElement>
  | { x: number; y: number }
  | undefined;

interface ThemeContextValue {
  theme: Theme;
  setTheme: (theme: Theme, origin?: ThemeTransitionOrigin) => void;
  toggleTheme: (origin?: ThemeTransitionOrigin) => void;
}

interface ViewTransitionLike {
  ready: Promise<void>;
  finished: Promise<void>;
}

type DocumentWithViewTransition = Document & {
  startViewTransition?: (callback: () => void) => ViewTransitionLike;
};

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

function resolveTransitionPoint(origin?: ThemeTransitionOrigin): {
  x: number;
  y: number;
} {
  if (typeof window === "undefined") {
    return { x: 0, y: 0 };
  }

  if (origin && "x" in origin && "y" in origin) {
    return { x: origin.x, y: origin.y };
  }

  if (origin && "currentTarget" in origin && origin.currentTarget) {
    const rect = origin.currentTarget.getBoundingClientRect();
    const hasPointerCoords =
      typeof origin.clientX === "number" &&
      typeof origin.clientY === "number" &&
      (origin.clientX > 0 || origin.clientY > 0);

    // For wide menu items, radiate from the tapped pointer coordinates;
    // for compact toggle buttons, radiate concentrically from button center.
    if (rect.width > 80 && hasPointerCoords) {
      return { x: origin.clientX, y: origin.clientY };
    }
    if (rect.width > 0 && rect.height > 0) {
      return {
        x: rect.left + rect.width / 2,
        y: rect.top + rect.height / 2,
      };
    }
    if (hasPointerCoords) {
      return { x: origin.clientX, y: origin.clientY };
    }
  }

  return { x: window.innerWidth - 36, y: 32 };
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

  useEffect(() => {
    const handleStorage = (event: StorageEvent) => {
      if (event.key !== THEME_STORAGE_KEY) {
        return;
      }
      if (event.newValue === "light" || event.newValue === "dark") {
        setLocalTheme(event.newValue);
      }
    };

    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  const theme: Theme =
    profile?.theme === "light" || profile?.theme === "dark"
      ? profile.theme
      : localTheme;

  useEffect(() => {
    if (!hydrated && !profile) {
      return;
    }
    document.documentElement.classList.toggle("dark", theme === "dark");
    try {
      localStorage.setItem(THEME_STORAGE_KEY, theme);
    } catch {
      // Ignore storage errors.
    }
  }, [hydrated, profile, theme]);

  const commitThemeState = useCallback(
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

  const setTheme = useCallback(
    (nextTheme: Theme, origin?: ThemeTransitionOrigin) => {
      if (typeof window === "undefined" || typeof document === "undefined") {
        commitThemeState(nextTheme);
        return;
      }

      const root = document.documentElement;
      const doc = document as DocumentWithViewTransition;
      const prefersReducedMotion =
        window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ??
        false;

      if (
        nextTheme === theme ||
        prefersReducedMotion ||
        typeof doc.startViewTransition !== "function"
      ) {
        root.classList.toggle("dark", nextTheme === "dark");
        commitThemeState(nextTheme);
        return;
      }

      const { x, y } = resolveTransitionPoint(origin);
      const endRadius = Math.hypot(
        Math.max(x, window.innerWidth - x),
        Math.max(y, window.innerHeight - y)
      );

      const transition = doc.startViewTransition(() => {
        root.classList.add("theme-transitioning");
        root.classList.toggle("dark", nextTheme === "dark");
        flushSync(() => {
          commitThemeState(nextTheme);
        });
        void root.offsetHeight;
      });

      transition.ready
        .then(() => {
          root.animate(
            {
              clipPath: [
                `circle(0px at ${x}px ${y}px)`,
                `circle(${endRadius}px at ${x}px ${y}px)`,
              ],
            },
            {
              duration: 520,
              easing: "cubic-bezier(0.22, 1, 0.36, 1)",
              pseudoElement: "::view-transition-new(root)",
            }
          );
        })
        .catch(() => {
          // Transition skipped or unsupported pseudo-element animation.
        });

      transition.finished
        .finally(() => {
          root.classList.remove("theme-transitioning");
        })
        .catch(() => {
          // Ignore rejected finished promise on interrupted transitions.
        });
    },
    [commitThemeState, theme]
  );

  const toggleTheme = useCallback(
    (origin?: ThemeTransitionOrigin) => {
      setTheme(theme === "light" ? "dark" : "light", origin);
    },
    [setTheme, theme]
  );

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
