"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useRouter } from "next/navigation";
import { apiClient, ApiError } from "@/lib/api";
import type { ProfileDTO } from "@/lib/types";

const TOKEN_STORAGE_KEY = "nouriko_access_token";

interface AuthContextValue {
  token: string | null;
  profile: ProfileDTO | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<ProfileDTO>;
  logout: () => Promise<void>;
  refreshProfile: () => Promise<ProfileDTO | null>;
  setProfile: (profile: ProfileDTO) => void;
  clearSession: (redirectToLogin?: boolean) => void;
  handleUnauthorized: (err: unknown) => boolean;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [token, setToken] = useState<string | null>(null);
  const [profile, setProfileState] = useState<ProfileDTO | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const tokenRef = useRef<string | null>(null);
  const sessionVersionRef = useRef<number>(0);

  const clearSession = useCallback(
    (redirectToLogin = false) => {
      sessionVersionRef.current += 1;
      tokenRef.current = null;
      try {
        sessionStorage.removeItem(TOKEN_STORAGE_KEY);
      } catch {
        // Ignore storage errors in restricted environments.
      }
      setToken(null);
      setProfileState(null);
      if (redirectToLogin) {
        router.replace("/login?reason=session_expired");
      }
    },
    [router]
  );

  const handleUnauthorized = useCallback(
    (err: unknown): boolean => {
      if (
        err instanceof ApiError &&
        (err.code === "UNAUTHORIZED" ||
          (err.status === 401 && err.code !== "INVALID_CREDENTIALS"))
      ) {
        clearSession(true);
        return true;
      }
      return false;
    },
    [clearSession]
  );

  useEffect(() => {
    let active = true;

    async function hydrateSession() {
      const version = sessionVersionRef.current;
      let storedToken: string | null = null;
      try {
        storedToken = sessionStorage.getItem(TOKEN_STORAGE_KEY);
      } catch {
        storedToken = null;
      }

      if (!storedToken) {
        if (active) {
          setIsLoading(false);
        }
        return;
      }

      try {
        const res = await apiClient.getProfile(storedToken);
        if (!active || sessionVersionRef.current !== version) return;
        tokenRef.current = storedToken;
        setToken(storedToken);
        setProfileState(res.profile);
      } catch (err) {
        if (!active || sessionVersionRef.current !== version) return;
        if (err instanceof ApiError && err.status === 401) {
          tokenRef.current = null;
          try {
            sessionStorage.removeItem(TOKEN_STORAGE_KEY);
          } catch {
            // Ignore storage errors.
          }
          setToken(null);
          setProfileState(null);
        }
      } finally {
        if (active) {
          setIsLoading(false);
        }
      }
    }

    void hydrateSession();

    return () => {
      active = false;
    };
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const res = await apiClient.login({ email, password });
    sessionVersionRef.current += 1;
    tokenRef.current = res.access_token;
    try {
      sessionStorage.setItem(TOKEN_STORAGE_KEY, res.access_token);
    } catch {
      // Ignore storage errors.
    }
    setToken(res.access_token);
    setProfileState(res.profile);
    return res.profile;
  }, []);

  const logout = useCallback(async () => {
    const currentToken = tokenRef.current;
    clearSession(false);
    if (currentToken) {
      try {
        await apiClient.logout(currentToken);
      } catch {
        // Logout is best-effort on server; local session is already cleared.
      }
    }
    router.push("/");
  }, [clearSession, router]);

  const refreshProfile = useCallback(async () => {
    const currentToken = tokenRef.current;
    if (!currentToken) return null;
    const version = sessionVersionRef.current;
    try {
      const res = await apiClient.getProfile(currentToken);
      if (sessionVersionRef.current !== version || !tokenRef.current) {
        return null;
      }
      setProfileState(res.profile);
      return res.profile;
    } catch (err) {
      if (sessionVersionRef.current === version) {
        handleUnauthorized(err);
      }
      return null;
    }
  }, [handleUnauthorized]);

  const setProfile = useCallback((nextProfile: ProfileDTO) => {
    if (!tokenRef.current) {
      return;
    }
    setProfileState(nextProfile);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      profile,
      isLoading,
      isAuthenticated: Boolean(token && profile),
      login,
      logout,
      refreshProfile,
      setProfile,
      clearSession,
      handleUnauthorized,
    }),
    [
      clearSession,
      handleUnauthorized,
      isLoading,
      login,
      logout,
      profile,
      refreshProfile,
      setProfile,
      token,
    ]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
