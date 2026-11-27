import type {
  AnalyzeChatParams,
  ChatAnalyzeResponse,
  CreateSavedMealRequest,
  CreateSavedMealResponse,
  DailySummaryResponse,
  DeleteAccountPayload,
  DeleteSavedMealResponse,
  GoogleLoginPayload,
  Language,
  LoginResponse,
  MealHistoryResponse,
  OnboardingPayload,
  ProfileResponse,
  RegisterResponse,
  StatusResponse,
  Theme,
  UpdateProfilePayload,
} from "@/lib/types";

const PUBLIC_API_BASE_URL = (process.env.NEXT_PUBLIC_API_URL ?? "").replace(
  /\/+$/,
  ""
);

function resolveApiUrl(path: string): string {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;

  if (typeof window === "undefined") {
    const internalBase = (
      process.env.API_INTERNAL_URL?.trim() ||
      PUBLIC_API_BASE_URL ||
      "http://localhost:8080"
    ).trim();
    const baseWithSlash = internalBase.endsWith("/")
      ? internalBase
      : `${internalBase}/`;
    return new URL(
      normalizedPath.replace(/^\/+/, ""),
      baseWithSlash
    ).toString();
  }

  if (!PUBLIC_API_BASE_URL) {
    return normalizedPath;
  }

  return `${PUBLIC_API_BASE_URL}${normalizedPath}`;
}

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
    response = await fetch(resolveApiUrl(path), {
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

  loginWithGoogle(payload: GoogleLoginPayload): Promise<LoginResponse> {
    return request<LoginResponse>({
      method: "POST",
      path: "/api/v1/auth/google",
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

  deleteAccount(
    token: string,
    payload: DeleteAccountPayload
  ): Promise<StatusResponse> {
    return request<StatusResponse>({
      method: "DELETE",
      path: "/api/v1/account",
      token,
      body: payload,
    });
  },

  async analyzeChat(
    token: string,
    params: AnalyzeChatParams
  ): Promise<ChatAnalyzeResponse> {
    const controller = new AbortController();
    let timedOut = false;
    const timeoutId = setTimeout(() => {
      timedOut = true;
      controller.abort();
    }, 65_000);

    const externalSignal = params.signal;
    const handleExternalAbort = () => controller.abort();
    if (externalSignal) {
      if (externalSignal.aborted) {
        clearTimeout(timeoutId);
        throw new ApiError("REQUEST_ABORTED", 0);
      }
      externalSignal.addEventListener("abort", handleExternalAbort, {
        once: true,
      });
    }

    const trimmedMessage = (params.message ?? "").trim();
    const cleanContext = (params.conversationContext ?? [])
      .slice(-6)
      .map((turn) => ({
        role: turn.role,
        summary: turn.summary.trim().slice(0, 300),
      }))
      .filter((turn) => turn.summary.length > 0);

    const headers: Record<string, string> = {
      Accept: "application/json",
      Authorization: `Bearer ${token}`,
    };

    let body: BodyInit;
    if (params.photo) {
      const formData = new FormData();
      if (trimmedMessage) {
        formData.append("message", trimmedMessage);
      }
      formData.append("language", params.language);
      if (cleanContext.length > 0) {
        formData.append("conversation_context", JSON.stringify(cleanContext));
      }
      formData.append("photo", params.photo, params.photo.name || "photo.jpg");
      body = formData;
    } else {
      headers["Content-Type"] = "application/json";
      body = JSON.stringify({
        message: trimmedMessage,
        language: params.language,
        conversation_context: cleanContext,
      });
    }

    let response: Response;
    try {
      response = await fetch(resolveApiUrl("/api/v1/chat/analyze"), {
        method: "POST",
        headers,
        body,
        cache: "no-store",
        signal: controller.signal,
      });
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") {
        if (!timedOut && externalSignal?.aborted) {
          throw new ApiError("REQUEST_ABORTED", 0);
        }
        throw new ApiError("AI_TIMEOUT", 0);
      }
      throw new ApiError("NETWORK_ERROR", 0);
    } finally {
      clearTimeout(timeoutId);
      externalSignal?.removeEventListener("abort", handleExternalAbort);
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
            : response.status === 403
              ? "ONBOARDING_REQUIRED"
              : response.status === 413
                ? "PAYLOAD_TOO_LARGE"
                : response.status === 422
                  ? "PHOTO_UNRECOGNIZED"
                  : response.status === 429
                    ? "TOO_MANY_REQUESTS"
                    : response.status === 503
                      ? "AI_UNAVAILABLE"
                      : response.status === 504
                        ? "AI_TIMEOUT"
                        : "INTERNAL_ERROR";
      throw new ApiError(code, response.status);
    }

    return data as ChatAnalyzeResponse;
  },

  saveMeal(
    token: string,
    payload: CreateSavedMealRequest
  ): Promise<CreateSavedMealResponse> {
    return request<CreateSavedMealResponse>({
      method: "POST",
      path: "/api/v1/meals",
      token,
      body: payload,
    });
  },

  deleteMeal(token: string, mealId: string): Promise<DeleteSavedMealResponse> {
    return request<DeleteSavedMealResponse>({
      method: "DELETE",
      path: `/api/v1/meals/${encodeURIComponent(mealId)}`,
      token,
    });
  },

  getDailySummary(token: string, date?: string): Promise<DailySummaryResponse> {
    const query = date ? `?date=${encodeURIComponent(date)}` : "";
    return request<DailySummaryResponse>({
      method: "GET",
      path: `/api/v1/meals/summary${query}`,
      token,
    });
  },

  getMealHistory(
    token: string,
    params?: { from?: string; to?: string; days?: number }
  ): Promise<MealHistoryResponse> {
    const search = new URLSearchParams();
    if (params?.from) {
      search.set("from", params.from);
    }
    if (params?.to) {
      search.set("to", params.to);
    }
    if (params?.days !== undefined) {
      search.set("days", String(params.days));
    }
    const qs = search.toString();
    return request<MealHistoryResponse>({
      method: "GET",
      path: `/api/v1/meals/history${qs ? `?${qs}` : ""}`,
      token,
    });
  },
};
