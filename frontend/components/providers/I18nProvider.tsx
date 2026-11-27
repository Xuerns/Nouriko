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
import { dictionaries, type Dictionary } from "@/lib/i18n";
import type { Language } from "@/lib/types";

const LANG_STORAGE_KEY = "nouriko_lang";

interface I18nContextValue {
  language: Language;
  t: Dictionary;
  hydrated: boolean;
  setLanguage: (lang: Language) => void;
  toggleLanguage: () => void;
}

const I18nContext = createContext<I18nContextValue | null>(null);

function detectInitialLanguage(): Language {
  if (typeof window === "undefined") {
    return "id";
  }
  try {
    const stored = localStorage.getItem(LANG_STORAGE_KEY);
    if (stored === "id" || stored === "en") {
      return stored;
    }
  } catch {
    // Ignore storage errors.
  }
  const browserLang = navigator.language?.toLowerCase() ?? "";
  if (browserLang.startsWith("en")) {
    return "en";
  }
  return "id";
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const { token, profile, setProfile, handleUnauthorized } = useAuth();
  const [localLanguage, setLocalLanguage] = useState<Language>("id");
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    const detected = detectInitialLanguage();
    const timer = window.setTimeout(() => {
      setLocalLanguage(detected);
      setHydrated(true);
    }, 0);
    return () => window.clearTimeout(timer);
  }, []);

  const language: Language =
    profile?.language === "id" || profile?.language === "en"
      ? profile.language
      : localLanguage;

  useEffect(() => {
    document.documentElement.lang = language;
    if (hydrated || profile) {
      try {
        localStorage.setItem(LANG_STORAGE_KEY, language);
      } catch {
        // Ignore storage errors.
      }
    }
  }, [hydrated, language, profile]);

  const setLanguage = useCallback(
    (nextLang: Language) => {
      setLocalLanguage(nextLang);
      try {
        localStorage.setItem(LANG_STORAGE_KEY, nextLang);
      } catch {
        // Ignore storage errors.
      }

      if (token && profile) {
        setProfile({ ...profile, language: nextLang });
        void apiClient
          .updateProfile(token, { language: nextLang })
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

  const toggleLanguage = useCallback(() => {
    setLanguage(language === "id" ? "en" : "id");
  }, [language, setLanguage]);

  const value = useMemo<I18nContextValue>(
    () => ({
      language,
      t: dictionaries[language],
      hydrated,
      setLanguage,
      toggleLanguage,
    }),
    [hydrated, language, setLanguage, toggleLanguage]
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useI18n must be used within I18nProvider");
  }
  return ctx;
}
