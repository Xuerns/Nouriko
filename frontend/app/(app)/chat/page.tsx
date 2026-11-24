"use client";

import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import {
  RotateCcw,
  SlidersHorizontal,
  SquarePen,
} from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/components/providers/AuthProvider";
import { useI18n } from "@/components/providers/I18nProvider";
import { Alert } from "@/components/ui/Alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ChatPageSkeleton } from "@/components/ui/PageSkeletons";
import { ChatComposer } from "@/components/chat/ChatComposer";
import {
  EmptyChatState,
  PromptSuggestions,
} from "@/components/chat/EmptyChatState";
import { MealResultCard } from "@/components/chat/MealResultCard";
import { PhotoCorrectionBar } from "@/components/chat/PhotoCorrectionBar";
import { ShoppingListCard } from "@/components/chat/ShoppingListCard";
import { ThinkingIndicator } from "@/components/chat/ThinkingIndicator";
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

function ChatContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialPrompt = searchParams.get("prompt") ?? "";
  const [prefillText] = useState(initialPrompt);
  const { token, isAuthenticated, isLoading, profile, handleUnauthorized } =
    useAuth();
  const { t, language } = useI18n();

  useEffect(() => {
    if (initialPrompt) {
      window.history.replaceState(null, "", window.location.pathname);
    }
  }, [initialPrompt]);

  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isSending, setIsSending] = useState(false);
  const [sendingWithPhoto, setSendingWithPhoto] = useState(false);
  const [composerErrorCode, setComposerErrorCode] = useState<string | null>(
    null
  );
  const [savedMeals, setSavedMeals] = useState<Record<string, boolean>>({});
  const [savingMeals, setSavingMeals] = useState<Record<string, boolean>>({});
  const [saveErrors, setSaveErrors] = useState<Record<string, string>>({});
  const [hasRetryableSubmission, setHasRetryableSubmission] = useState(false);

  const lastSubmissionRef = useRef<PendingSubmission | null>(null);
  const activeAbortRef = useRef<AbortController | null>(null);
  const messagesRef = useRef<ChatMessage[]>([]);
  const savingMealsRef = useRef<Record<string, boolean>>({});
  const savedMealsRef = useRef<Record<string, boolean>>({});
  const bottomAnchorRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  useEffect(() => {
    return () => {
      activeAbortRef.current?.abort();
      activeAbortRef.current = null;
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

      activeAbortRef.current?.abort();
      const abortController = new AbortController();
      activeAbortRef.current = abortController;

      setIsSending(true);
      setSendingWithPhoto(Boolean(submission.photo));
      setComposerErrorCode(null);

      try {
        const result = await apiClient.analyzeChat(token, {
          message: submission.message,
          photo: submission.photo,
          language,
          conversationContext: contextSnapshot,
          signal: abortController.signal,
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
        if (err instanceof ApiError && err.code === "REQUEST_ABORTED") {
          return;
        }
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
        if (activeAbortRef.current === abortController) {
          activeAbortRef.current = null;
        }
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

  const handleStop = useCallback(() => {
    if (!activeAbortRef.current) return;
    activeAbortRef.current.abort();
    activeAbortRef.current = null;
  }, []);

  const handleClearChat = useCallback(() => {
    if (isSending) return;
    activeAbortRef.current?.abort();
    activeAbortRef.current = null;
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
    setComposerErrorCode(null);
  }, [isSending]);

  const handleSaveMeal = useCallback(
    async (
      key: string,
      meal: MealOption,
      mealTime?: MealTime,
      mealDate?: string
    ) => {
      if (!token || savingMealsRef.current[key] || savedMealsRef.current[key]) {
        return;
      }

      const targetDate = mealDate || getLocalDateString();

      savingMealsRef.current[key] = true;
      setSavingMeals((prev) => ({ ...prev, [key]: true }));
      setSaveErrors((prev) => {
        const next = { ...prev };
        delete next[key];
        return next;
      });

      try {
        await apiClient.saveMeal(token, {
          meal_date: targetDate,
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
        toast.success(t.chat.saveMealFeedback, {
          action: {
            label: t.chat.viewSummaryLink,
            onClick: () =>
              router.push(`/summary?date=${encodeURIComponent(targetDate)}`),
          },
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
    [
      handleUnauthorized,
      language,
      router,
      t.chat.saveMealFeedback,
      t.chat.viewSummaryLink,
      token,
    ]
  );

  if (
    isLoading ||
    !isAuthenticated ||
    !profile ||
    !profile.onboarding_completed
  ) {
    return <ChatPageSkeleton label={t.common.loading} />;
  }

  return (
    <div className="flex h-full min-h-0 flex-col gap-4">
      <div className="shrink-0 flex flex-wrap items-center justify-between gap-3 border-b border-border pb-3.5">
        <div className="min-w-0 space-y-0.5">
          <h1 className="text-base font-semibold leading-tight tracking-[-0.01em] text-foreground">
            {t.chat.title}
          </h1>
          <p className="text-xs leading-tight text-muted-foreground">
            {t.chat.targetSummary}:{" "}
            <span className="font-semibold tabular-nums text-[var(--brand-primary)]">
              {profile.daily_calorie_target ?? "-"} {t.profile.kcalUnit}
            </span>
          </p>
        </div>

        <div className="flex items-center gap-2">
          {messages.length > 0 ? (
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={isSending}
              onClick={handleClearChat}
              className="h-9 rounded-full bg-[var(--brand-tint)] px-3.5 text-xs font-medium text-[var(--brand-primary)] shadow-none hover:bg-[var(--brand-tint)]/80"
            >
              <SquarePen
                className="size-3.5 shrink-0 stroke-[1.75]"
                aria-hidden="true"
              />
              <span>{t.chat.clearChatButton}</span>
            </Button>
          ) : null}

          <Button
            render={<Link href="/profile" />}
            variant="outline"
            size="sm"
            className="h-9 rounded-full border-border bg-card px-3.5 text-xs font-medium text-muted-foreground shadow-none hover:bg-accent hover:text-foreground"
          >
            <SlidersHorizontal
              className="size-3.5 shrink-0 stroke-[1.75]"
              aria-hidden="true"
            />
            <span>{t.chat.editProfileLink}</span>
          </Button>
        </div>
      </div>

      <section
        aria-label={t.chat.title}
        className="no-scrollbar min-h-0 flex-1 overflow-y-auto space-y-4"
      >
        {messages.length === 0 && !isSending ? (
          <EmptyChatState displayName={profile.display_name} />
        ) : (
          <div className="space-y-5">
            {messages.map((msg, msgIdx) => {
              if (msg.role === "user") {
                return (
                  <div key={msg.id} className="flex justify-end">
                    <div className="max-w-[85%] rounded-3xl rounded-br-md bg-muted px-4 py-3.5 space-y-2.5 sm:max-w-[78%]">
                      {msg.photoPreviewUrl ? (
                        <div>
                          {/* eslint-disable-next-line @next/next/no-img-element */}
                          <img
                            src={msg.photoPreviewUrl}
                            alt={msg.photoName || t.chat.photoAttachedFallback}
                            className="max-h-56 w-auto rounded-2xl border border-border object-contain"
                          />
                        </div>
                      ) : null}
                      <p className="text-sm leading-relaxed text-foreground whitespace-pre-wrap">
                        {msg.text || t.chat.photoAttachedFallback}
                      </p>
                    </div>
                  </div>
                );
              }

              if (msg.errorCode) {
                return (
                  <div key={msg.id} className="space-y-2">
                    <div className="flex items-center gap-2 px-1">
                      <Image
                        src="/logo.png"
                        alt=""
                        width={20}
                        height={20}
                        aria-hidden="true"
                        className="size-5 shrink-0 object-contain"
                      />
                      <span className="text-xs font-semibold text-foreground">
                        {t.chat.assistantLabel}
                      </span>
                    </div>
                    <Card className="rounded-3xl rounded-bl-md border border-border bg-card p-4 shadow-none ring-0 gap-3">
                      <Alert variant="error">
                        {getFriendlyErrorMessage(msg.errorCode, language)}
                      </Alert>
                      {hasRetryableSubmission &&
                      msgIdx === messages.length - 1 ? (
                        <div>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={isSending}
                            onClick={() => handleRetry(msg.id)}
                            className="h-9 rounded-full px-4 text-xs shadow-none"
                          >
                            <RotateCcw
                              className="size-3.5 shrink-0 stroke-[1.75]"
                              aria-hidden="true"
                            />
                            <span>{t.chat.retryButton}</span>
                          </Button>
                        </div>
                      ) : null}
                    </Card>
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
                <div key={msg.id} className="space-y-3">
                  <div className="flex items-center gap-2 px-1">
                    <Image
                      src="/logo.png"
                      alt=""
                      width={20}
                      height={20}
                      aria-hidden="true"
                      className="size-5 shrink-0 object-contain"
                    />
                    <span className="text-xs font-semibold text-foreground">
                      {t.chat.assistantLabel}
                    </span>
                  </div>

                  {resp.intro ? (
                    <p className="px-1 text-[15px] leading-[1.6] text-foreground">
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
                            onSave={(selectedMeal, mealTime, mealDate) =>
                              handleSaveMeal(
                                key,
                                selectedMeal,
                                mealTime,
                                mealDate
                              )
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
              <div className="space-y-1.5">
                <div className="flex items-center gap-2 px-1">
                  <Image
                    src="/logo.png"
                    alt=""
                    width={20}
                    height={20}
                    aria-hidden="true"
                    className="size-5 shrink-0 object-contain"
                  />
                  <span className="text-xs font-semibold text-foreground">
                    {t.chat.assistantLabel}
                  </span>
                </div>
                <ThinkingIndicator
                  key={sendingWithPhoto ? "photo" : "text"}
                  steps={
                    sendingWithPhoto
                      ? t.chat.photoThinkingSteps
                      : t.chat.thinkingSteps
                  }
                />
              </div>
            ) : null}

            <div ref={bottomAnchorRef} />
          </div>
        )}
      </section>

      <div className="shrink-0 min-w-0 space-y-2.5">
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
          prefillMessage={prefillText}
          onSend={handleSend}
          onStop={handleStop}
          disabled={isSending}
          onPhotoError={(code) => setComposerErrorCode(code)}
        />
      </div>
    </div>
  );
}

export default function ChatPage() {
  return (
    <Suspense fallback={<ChatPageSkeleton />}>
      <ChatContent />
    </Suspense>
  );
}
