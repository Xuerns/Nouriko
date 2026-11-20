"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Loader2, RotateCcw } from "lucide-react";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { ChatComposer } from "@/components/chat/ChatComposer";
import {
  EmptyChatState,
  PromptSuggestions,
} from "@/components/chat/EmptyChatState";
import { MealResultCard } from "@/components/chat/MealResultCard";
import { PhotoCorrectionBar } from "@/components/chat/PhotoCorrectionBar";
import { ShoppingListCard } from "@/components/chat/ShoppingListCard";
import { apiClient, ApiError } from "@/lib/api";
import { getLocalDateString } from "@/lib/date";
import { getFriendlyErrorMessage } from "@/lib/errors";
import type {
  ChatAnalyzeResponse,
  ChatMessage,
  ConversationTurn,
  MealOption,
  MealTime,
} from "@/lib/types";

interface PendingSubmission {
  message: string;
  photo: File | null;
}

function createMessageId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `msg-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

function summarizeAssistantResponse(resp: ChatAnalyzeResponse): string {
  const parts: string[] = [];
  if (resp.intro) {
    parts.push(resp.intro.trim());
  }
  if (resp.detected_items && resp.detected_items.length > 0) {
    parts.push(`Items: ${resp.detected_items.join(", ")}`);
  }
  if (resp.meals && resp.meals.length > 0) {
    parts.push(`Meals: ${resp.meals.map((m) => m.name).join(", ")}`);
  }
  return parts.join(" | ").slice(0, 300);
}

function buildConversationContext(messages: ChatMessage[]): ConversationTurn[] {
  const turns: ConversationTurn[] = [];
  for (const msg of messages) {
    if (msg.role === "user") {
      const summary = (
        msg.text.trim() || (msg.hasPhoto ? "Uploaded a food photo" : "")
      ).slice(0, 300);
      if (summary) {
        turns.push({ role: "user", summary });
      }
    } else if (msg.response) {
      const summary = summarizeAssistantResponse(msg.response);
      if (summary) {
        turns.push({ role: "assistant", summary });
      }
    }
  }
  return turns.slice(-6);
}

function mealKey(messageId: string, meal: MealOption, idx: number): string {
  return `${messageId}:${idx}:${meal.name}`;
}

export default function ChatPage() {
  const router = useRouter();
  const { token, isAuthenticated, isLoading, profile, handleUnauthorized } =
    useAuth();
  const { t, language } = useI18n();

  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isSending, setIsSending] = useState(false);
  const [sendingWithPhoto, setSendingWithPhoto] = useState(false);
  const [composerErrorCode, setComposerErrorCode] = useState<string | null>(
    null
  );
  const [savedMeals, setSavedMeals] = useState<Record<string, boolean>>({});
  const [savingMeals, setSavingMeals] = useState<Record<string, boolean>>({});
  const [saveErrors, setSaveErrors] = useState<Record<string, string>>({});
  const [saveNotice, setSaveNotice] = useState(false);
  const [hasRetryableSubmission, setHasRetryableSubmission] = useState(false);

  const lastSubmissionRef = useRef<PendingSubmission | null>(null);
  const messagesRef = useRef<ChatMessage[]>([]);
  const savingMealsRef = useRef<Record<string, boolean>>({});
  const savedMealsRef = useRef<Record<string, boolean>>({});
  const bottomAnchorRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  useEffect(() => {
    return () => {
      for (const msg of messagesRef.current) {
        if (msg.photoPreviewUrl) {
          URL.revokeObjectURL(msg.photoPreviewUrl);
        }
      }
    };
  }, []);

  useEffect(() => {
    if (isLoading) return;
    if (!isAuthenticated) {
      router.replace("/login");
      return;
    }
    if (profile && !profile.onboarding_completed) {
      router.replace("/onboarding");
    }
  }, [isAuthenticated, isLoading, profile, router]);

  useEffect(() => {
    if (messages.length > 0 || isSending) {
      bottomAnchorRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages.length, isSending]);

  const executeAnalyze = useCallback(
    async (
      submission: PendingSubmission,
      contextSnapshot: ConversationTurn[],
      replaceLastErrorId?: string
    ) => {
      if (!token) return;

      setIsSending(true);
      setSendingWithPhoto(Boolean(submission.photo));
      setComposerErrorCode(null);

      try {
        const result = await apiClient.analyzeChat(token, {
          message: submission.message,
          photo: submission.photo,
          language,
          conversationContext: contextSnapshot,
        });

        const assistantMsg: ChatMessage = {
          id: createMessageId(),
          role: "assistant",
          text: result.intro,
          response: result,
          hasPhoto: Boolean(submission.photo),
          createdAt: Date.now(),
        };

        setMessages((prev) => {
          const filtered = replaceLastErrorId
            ? prev.filter((m) => m.id !== replaceLastErrorId)
            : prev;
          return [...filtered, assistantMsg];
        });
      } catch (err) {
        if (handleUnauthorized(err)) {
          return;
        }
        if (err instanceof ApiError && err.code === "ONBOARDING_REQUIRED") {
          router.replace("/onboarding");
          return;
        }

        const errorCode =
          err instanceof ApiError ? err.code : "INTERNAL_ERROR";

        const errorMsg: ChatMessage = {
          id: createMessageId(),
          role: "assistant",
          text: "",
          errorCode,
          hasPhoto: Boolean(submission.photo),
          createdAt: Date.now(),
        };

        setMessages((prev) => {
          const filtered = replaceLastErrorId
            ? prev.filter((m) => m.id !== replaceLastErrorId)
            : prev;
          return [...filtered, errorMsg];
        });
      } finally {
        setIsSending(false);
        setSendingWithPhoto(false);
      }
    },
    [handleUnauthorized, language, router, token]
  );

  const handleSend = useCallback(
    (payload: {
      message: string;
      photo: File | null;
      photoPreviewUrl?: string;
    }) => {
      if (isSending) return;
      const trimmed = payload.message.trim();
      if (!trimmed && !payload.photo) return;

      const contextSnapshot = buildConversationContext(messagesRef.current);
      const userMsg: ChatMessage = {
        id: createMessageId(),
        role: "user",
        text: trimmed,
        photoPreviewUrl: payload.photoPreviewUrl,
        photoName: payload.photo?.name,
        hasPhoto: Boolean(payload.photo),
        createdAt: Date.now(),
      };

      const submission: PendingSubmission = {
        message: trimmed,
        photo: payload.photo,
      };
      lastSubmissionRef.current = submission;
      setHasRetryableSubmission(true);

      setMessages((prev) => [...prev, userMsg]);
      void executeAnalyze(submission, contextSnapshot);
    },
    [executeAnalyze, isSending]
  );

  const handlePromptSelect = useCallback(
    (prompt: string) => {
      handleSend({ message: prompt, photo: null });
    },
    [handleSend]
  );

  const handleRetry = useCallback(
    (failedMessageId: string) => {
      if (isSending || !lastSubmissionRef.current) return;
      const withoutFailed = messagesRef.current.filter(
        (m) => m.id !== failedMessageId
      );
      // Exclude the last user turn that matches the retried submission so it is not duplicated in context.
      const contextBase =
        withoutFailed.length > 0 &&
        withoutFailed[withoutFailed.length - 1].role === "user"
          ? withoutFailed.slice(0, -1)
          : withoutFailed;

      const contextSnapshot = buildConversationContext(contextBase);
      void executeAnalyze(
        lastSubmissionRef.current,
        contextSnapshot,
        failedMessageId
      );
    },
    [executeAnalyze, isSending]
  );

  const handleClearChat = useCallback(() => {
    if (isSending) return;
    for (const msg of messagesRef.current) {
      if (msg.photoPreviewUrl) {
        URL.revokeObjectURL(msg.photoPreviewUrl);
      }
    }
    lastSubmissionRef.current = null;
    savingMealsRef.current = {};
    savedMealsRef.current = {};
    setHasRetryableSubmission(false);
    setMessages([]);
    setSavedMeals({});
    setSavingMeals({});
    setSaveErrors({});
    setSaveNotice(false);
    setComposerErrorCode(null);
  }, [isSending]);

  const handleSaveMeal = useCallback(
    async (key: string, meal: MealOption, mealTime?: MealTime) => {
      if (!token || savingMealsRef.current[key] || savedMealsRef.current[key]) {
        return;
      }

      savingMealsRef.current[key] = true;
      setSavingMeals((prev) => ({ ...prev, [key]: true }));
      setSaveErrors((prev) => {
        const next = { ...prev };
        delete next[key];
        return next;
      });

      try {
        await apiClient.saveMeal(token, {
          meal_date: getLocalDateString(),
          ...(mealTime ? { meal_time: mealTime } : {}),
          name: meal.name,
          health_score: Math.max(
            0,
            Math.min(100, Math.round(meal.health_score))
          ),
          calories: Math.max(0, Math.round(meal.calories)),
          protein_g: Math.max(0, meal.protein_g),
          carbs_g: Math.max(0, meal.carbs_g),
          fat_g: Math.max(0, meal.fat_g),
          fiber_g: Math.max(0, meal.fiber_g),
          price_idr:
            meal.estimated_price_idr > 0
              ? Math.round(meal.estimated_price_idr)
              : null,
          input_source: meal.input_source,
        });

        savedMealsRef.current[key] = true;
        setSavedMeals((prev) => ({ ...prev, [key]: true }));
        setSaveNotice(true);
      } catch (err) {
        if (handleUnauthorized(err)) {
          return;
        }
        if (err instanceof ApiError && err.code === "ONBOARDING_REQUIRED") {
          router.replace("/onboarding");
          return;
        }
        const errorCode =
          err instanceof ApiError && err.code === "TOO_MANY_REQUESTS"
            ? "TOO_MANY_REQUESTS"
            : "MEAL_SAVE_FAILED";
        setSaveErrors((prev) => ({
          ...prev,
          [key]: getFriendlyErrorMessage(errorCode, language),
        }));
      } finally {
        delete savingMealsRef.current[key];
        setSavingMeals((prev) => {
          const next = { ...prev };
          delete next[key];
          return next;
        });
      }
    },
    [handleUnauthorized, language, router, token]
  );

  if (
    isLoading ||
    !isAuthenticated ||
    !profile ||
    !profile.onboarding_completed
  ) {
    return (
      <div className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-6 text-sm text-neutral-600 dark:text-neutral-400">
        {t.common.loading}
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col gap-4">
      <section className="shrink-0 rounded-lg border border-neutral-300 dark:border-neutral-700 p-3.5 sm:p-4 flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-1">
          <h1 className="text-xl font-semibold text-neutral-900 dark:text-neutral-100">
            {t.chat.title}
          </h1>
          <p className="text-sm text-neutral-600 dark:text-neutral-400">
            {t.chat.targetSummary}:{" "}
            <span className="font-mono font-medium text-neutral-900 dark:text-neutral-100">
              {profile.daily_calorie_target ?? "-"} {t.profile.kcalUnit}
            </span>
          </p>
          <p className="text-xs text-neutral-500 dark:text-neutral-400">
            {t.chat.ephemeralNotice}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {messages.length > 0 ? (
            <button
              type="button"
              disabled={isSending}
              onClick={handleClearChat}
              className="rounded-md border border-neutral-300 dark:border-neutral-700 px-3 py-2 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
            >
              {t.chat.clearChatButton}
            </button>
          ) : null}

          <Link
            href="/profile"
            className="rounded-md border border-neutral-300 dark:border-neutral-700 px-3.5 py-2 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
          >
            {t.chat.editProfileLink}
          </Link>
        </div>
      </section>

      {saveNotice ? (
        <Alert variant="success">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span>{t.chat.saveMealFeedback}</span>
            <Link
              href="/summary"
              className="text-xs font-medium underline underline-offset-4 text-neutral-900 dark:text-neutral-100"
            >
              {t.chat.viewSummaryLink}
            </Link>
          </div>
        </Alert>
      ) : null}

      <section
        aria-label={t.chat.title}
        className="min-h-0 flex-1 overflow-y-auto space-y-4 pr-1"
      >
        {messages.length === 0 && !isSending ? (
          <EmptyChatState />
        ) : (
          <div className="space-y-4">
            {messages.map((msg, msgIdx) => {
              if (msg.role === "user") {
                return (
                  <div key={msg.id} className="flex justify-end">
                    <div className="w-full max-w-2xl rounded-lg border border-neutral-400 dark:border-neutral-600 p-4 space-y-2.5">
                      <p className="font-mono text-xs uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
                        {t.chat.youLabel}
                      </p>
                      {msg.photoPreviewUrl ? (
                        <div>
                          {/* eslint-disable-next-line @next/next/no-img-element */}
                          <img
                            src={msg.photoPreviewUrl}
                            alt={msg.photoName || t.chat.photoAttachedFallback}
                            className="max-h-56 w-auto rounded-md border border-neutral-300 dark:border-neutral-700 object-contain"
                          />
                        </div>
                      ) : null}
                      <p className="text-sm text-neutral-900 dark:text-neutral-100 whitespace-pre-wrap leading-relaxed">
                        {msg.text || t.chat.photoAttachedFallback}
                      </p>
                    </div>
                  </div>
                );
              }

              if (msg.errorCode) {
                return (
                  <div
                    key={msg.id}
                    className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 space-y-3"
                  >
                    <p className="font-mono text-xs uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
                      {t.chat.assistantLabel}
                    </p>
                    <Alert variant="error">
                      {getFriendlyErrorMessage(msg.errorCode, language)}
                    </Alert>
                    {hasRetryableSubmission && msgIdx === messages.length - 1 ? (
                      <div>
                        <button
                          type="button"
                          disabled={isSending}
                          onClick={() => handleRetry(msg.id)}
                          className="inline-flex items-center gap-1.5 rounded-md border border-neutral-300 dark:border-neutral-700 px-3 py-1.5 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
                        >
                          <RotateCcw
                            className="h-3.5 w-3.5 shrink-0"
                            aria-hidden="true"
                          />
                          <span>{t.chat.retryButton}</span>
                        </button>
                      </div>
                    ) : null}
                  </div>
                );
              }

              const resp = msg.response;
              if (!resp) return null;

              const showPhotoCorrection =
                resp.needs_clarification ||
                resp.intent === "photo_food" ||
                resp.intent === "photo_ingredients" ||
                Boolean(msg.hasPhoto);

              const showShoppingList =
                resp.intent === "budget_ingredients" &&
                Array.isArray(resp.shopping_list) &&
                resp.shopping_list.length > 0;

              return (
                <div
                  key={msg.id}
                  className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 space-y-4"
                >
                  <p className="font-mono text-xs uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
                    {t.chat.assistantLabel}
                  </p>

                  {resp.intro ? (
                    <p className="text-sm text-neutral-900 dark:text-neutral-100 leading-relaxed">
                      {resp.intro}
                    </p>
                  ) : null}

                  {resp.off_topic_notice ? (
                    <Alert variant="info">{resp.off_topic_notice}</Alert>
                  ) : null}

                  {resp.budget_exceeded_notice ? (
                    <Alert variant="info">{resp.budget_exceeded_notice}</Alert>
                  ) : null}

                  {showShoppingList ? (
                    <ShoppingListCard items={resp.shopping_list} />
                  ) : null}

                  {Array.isArray(resp.meals) && resp.meals.length > 0 ? (
                    <div className="space-y-3">
                      {resp.meals.map((meal, idx) => {
                        const key = mealKey(msg.id, meal, idx);
                        return (
                          <MealResultCard
                            key={key}
                            meal={meal}
                            saved={Boolean(savedMeals[key])}
                            isSaving={Boolean(savingMeals[key])}
                            saveError={saveErrors[key] ?? null}
                            onSave={(selectedMeal, mealTime) =>
                              handleSaveMeal(key, selectedMeal, mealTime)
                            }
                          />
                        );
                      })}
                    </div>
                  ) : null}

                  {showPhotoCorrection ? (
                    <PhotoCorrectionBar
                      intent={resp.intent}
                      detectedItems={resp.detected_items ?? []}
                      needsClarification={Boolean(resp.needs_clarification)}
                      defaultFoodName={resp.meals?.[0]?.name ?? ""}
                      disabled={isSending}
                      onSubmitCorrection={handlePromptSelect}
                    />
                  ) : null}
                </div>
              );
            })}

            {isSending ? (
              <div
                role="status"
                aria-live="polite"
                className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 flex items-center gap-2.5 text-sm text-neutral-700 dark:text-neutral-300"
              >
                <Loader2
                  className="h-4 w-4 shrink-0 animate-spin"
                  aria-hidden="true"
                />
                <span>
                  {sendingWithPhoto
                    ? t.chat.analyzingPhotoText
                    : t.chat.analyzingText}
                </span>
              </div>
            ) : null}

            <div ref={bottomAnchorRef} />
          </div>
        )}
      </section>

      <div className="shrink-0 space-y-2">
        {messages.length === 0 && !isSending ? (
          <PromptSuggestions
            onSelectPrompt={handlePromptSelect}
            disabled={isSending}
          />
        ) : null}

        {composerErrorCode ? (
          <Alert variant="error">
            {getFriendlyErrorMessage(composerErrorCode, language)}
          </Alert>
        ) : null}

        <ChatComposer
          onSend={handleSend}
          disabled={isSending}
          onPhotoError={(code) => setComposerErrorCode(code)}
        />
      </div>
    </div>
  );
}
