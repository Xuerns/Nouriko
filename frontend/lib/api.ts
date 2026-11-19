import type {
  Language,
  LoginResponse,
  OnboardingPayload,
  ProfileResponse,
  RegisterResponse,
  StatusResponse,
  Theme,
  UpdateProfilePayload,
} from "@/lib/types";

const API_BASE_URL = (
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
).replace(/\/+$/, "");

export class ApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(code: string, status: number) {
    super(code);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

interface RequestOptions {
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  path: string;
  token?: string | null;
  body?: unknown;
}

async function request<T>({
  method,
  path,
  token,
  body,
}: RequestOptions): Promise<T> {
  const headers: Record<string, string> = {
    Accept: "application/json",
  };

  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      cache: "no-store",
    });
  } catch {
    throw new ApiError("NETWORK_ERROR", 0);
  }

  let data: unknown = null;
  try {
    data = await response.json();
  } catch {
    data = null;
  }

  if (!response.ok) {
    const code =
      typeof data === "object" &&
      data !== null &&
      "code" in data &&
      typeof (data as { code: unknown }).code === "string"
        ? (data as { code: string }).code
        : response.status === 401
          ? "UNAUTHORIZED"
          : response.status === 429
            ? "TOO_MANY_REQUESTS"
            : "INTERNAL_ERROR";
    throw new ApiError(code, response.status);
  }

  return data as T;
}

export const apiClient = {
  register(payload: {
    email: string;
    password: string;
    language: Language;
    theme: Theme;
  }): Promise<RegisterResponse> {
    return request<RegisterResponse>({
      method: "POST",
      path: "/api/v1/auth/register",
      body: payload,
    });
  },

  verifyEmail(token: string): Promise<StatusResponse> {
    return request<StatusResponse>({
      method: "POST",
      path: "/api/v1/auth/verify-email",
      body: { token },
    });
  },

  login(payload: {
    email: string;
    password: string;
  }): Promise<LoginResponse> {
    return request<LoginResponse>({
      method: "POST",
      path: "/api/v1/auth/login",
      body: payload,
    });
  },

  logout(token: string): Promise<StatusResponse> {
    return request<StatusResponse>({
      method: "POST",
      path: "/api/v1/auth/logout",
      token,
    });
  },

  forgotPassword(email: string): Promise<StatusResponse> {
    return request<StatusResponse>({
      method: "POST",
      path: "/api/v1/auth/forgot-password",
      body: { email },
    });
  },

  resetPassword(payload: {
    token: string;
    new_password: string;
  }): Promise<StatusResponse> {
    return request<StatusResponse>({
      method: "POST",
      path: "/api/v1/auth/reset-password",
      body: payload,
    });
  },

  changePassword(
    token: string,
    payload: { current_password: string; new_password: string }
  ): Promise<StatusResponse> {
    return request<StatusResponse>({
      method: "POST",
      path: "/api/v1/auth/change-password",
      token,
      body: payload,
    });
  },

  getProfile(token: string): Promise<ProfileResponse> {
    return request<ProfileResponse>({
      method: "GET",
      path: "/api/v1/profile",
      token,
    });
  },

  completeOnboarding(
    token: string,
    payload: OnboardingPayload
  ): Promise<ProfileResponse> {
    return request<ProfileResponse>({
      method: "PUT",
      path: "/api/v1/profile/onboarding",
      token,
      body: payload,
    });
  },

  updateProfile(
    token: string,
    payload: UpdateProfilePayload
  ): Promise<ProfileResponse> {
    return request<ProfileResponse>({
      method: "PATCH",
      path: "/api/v1/profile",
      token,
      body: payload,
    });
  },

  deleteAccount(token: string, password: string): Promise<StatusResponse> {
    return request<StatusResponse>({
      method: "DELETE",
      path: "/api/v1/account",
      token,
      body: { password },
    });
  },
};
