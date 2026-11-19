"use client";

import {
  useEffect,
  useRef,
  useState,
  type ChangeEvent,
  type FormEvent,
  type KeyboardEvent,
} from "react";
import { Camera, Loader2, Send, X } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { ApiError } from "@/lib/api";
import { ACCEPTED_PHOTO_INPUT, preparePhotoForUpload } from "@/lib/image";

interface ChatComposerProps {
  onSend: (payload: {
    message: string;
    photo: File | null;
    photoPreviewUrl?: string;
  }) => void;
  disabled?: boolean;
  onPhotoError?: (code: string) => void;
}

export function ChatComposer({
  onSend,
  disabled = false,
  onPhotoError,
}: ChatComposerProps) {
  const { t } = useI18n();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [message, setMessage] = useState("");
  const [photoFile, setPhotoFile] = useState<File | null>(null);
  const [photoPreviewUrl, setPhotoPreviewUrl] = useState<string | null>(null);
  const [isPreparingPhoto, setIsPreparingPhoto] = useState(false);

  const mountedRef = useRef(true);
  const previewUrlRef = useRef<string | null>(null);
  useEffect(() => {
    previewUrlRef.current = photoPreviewUrl;
  }, [photoPreviewUrl]);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      if (previewUrlRef.current) {
        URL.revokeObjectURL(previewUrlRef.current);
        previewUrlRef.current = null;
      }
    };
  }, []);

  async function handleFileChange(e: ChangeEvent<HTMLInputElement>) {
    const selected = e.target.files?.[0];
    e.target.value = "";
    if (!selected) return;

    setIsPreparingPhoto(true);
    try {
      const prepared = await preparePhotoForUpload(selected);
      if (!mountedRef.current) {
        return;
      }
      if (previewUrlRef.current) {
        URL.revokeObjectURL(previewUrlRef.current);
      }
      const nextPreviewUrl = URL.createObjectURL(prepared);
      previewUrlRef.current = nextPreviewUrl;
      setPhotoFile(prepared);
      setPhotoPreviewUrl(nextPreviewUrl);
    } catch (err) {
      if (!mountedRef.current) {
        return;
      }
      const code = err instanceof ApiError ? err.code : "PHOTO_INVALID";
      onPhotoError?.(code);
    } finally {
      if (mountedRef.current) {
        setIsPreparingPhoto(false);
      }
    }
  }

  function handleRemovePhoto() {
    if (photoPreviewUrl) {
      URL.revokeObjectURL(photoPreviewUrl);
    }
    setPhotoFile(null);
    setPhotoPreviewUrl(null);
  }

  function triggerSend() {
    if (disabled || isPreparingPhoto) return;
    const trimmed = message.trim();
    if (!trimmed && !photoFile) return;

    // Transfer ownership of the preview URL to the chat message list so it stays visible in history.
    const nextPreview = photoPreviewUrl ?? undefined;
    const nextPhoto = photoFile;

    previewUrlRef.current = null;
    setMessage("");
    setPhotoFile(null);
    setPhotoPreviewUrl(null);

    onSend({
      message: trimmed,
      photo: nextPhoto,
      photoPreviewUrl: nextPreview,
    });
  }

  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    triggerSend();
  }

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      triggerSend();
    }
  }

  const canSend =
    !disabled &&
    !isPreparingPhoto &&
    (message.trim().length > 0 || photoFile !== null);

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-3 space-y-3"
    >
      {photoPreviewUrl && photoFile ? (
        <div className="flex items-center justify-between gap-3 rounded-md border border-neutral-300 dark:border-neutral-700 p-2">
          <div className="flex items-center gap-3 min-w-0">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={photoPreviewUrl}
              alt={photoFile.name}
              className="h-12 w-12 shrink-0 rounded-md border border-neutral-300 dark:border-neutral-700 object-cover"
            />
            <div className="min-w-0">
              <p className="truncate text-xs font-medium text-neutral-900 dark:text-neutral-100">
                {photoFile.name}
              </p>
              <p className="font-mono text-xs text-neutral-500 dark:text-neutral-400">
                {Math.max(1, Math.round(photoFile.size / 1024))} KB
              </p>
            </div>
          </div>

          <button
            type="button"
            onClick={handleRemovePhoto}
            disabled={disabled}
            aria-label={t.chat.removePhotoLabel}
            className="inline-flex items-center gap-1 rounded-md border border-neutral-300 dark:border-neutral-700 px-2 py-1 text-xs text-neutral-700 dark:text-neutral-300 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
          >
            <X className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            <span>{t.chat.removePhotoLabel}</span>
          </button>
        </div>
      ) : null}

      {isPreparingPhoto ? (
        <div className="flex items-center gap-2 text-xs text-neutral-600 dark:text-neutral-400">
          <Loader2
            className="h-3.5 w-3.5 shrink-0 animate-spin"
            aria-hidden="true"
          />
          <span>{t.chat.processingPhoto}</span>
        </div>
      ) : null}

      <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
        <div className="flex flex-1 items-end gap-2">
          <input
            ref={fileInputRef}
            type="file"
            accept={ACCEPTED_PHOTO_INPUT}
            onChange={handleFileChange}
            className="sr-only"
            aria-label={t.chat.attachPhotoLabel}
          />

          <button
            type="button"
            disabled={disabled || isPreparingPhoto}
            onClick={() => fileInputRef.current?.click()}
            aria-label={t.chat.attachPhotoLabel}
            title={t.chat.attachPhotoLabel}
            className="inline-flex h-10 shrink-0 items-center justify-center gap-1.5 rounded-md border border-neutral-300 dark:border-neutral-700 px-3 text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
          >
            <Camera className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span className="sr-only sm:not-sr-only">
              {t.chat.attachPhotoLabel}
            </span>
          </button>

          <textarea
            rows={1}
            maxLength={1500}
            value={message}
            disabled={disabled}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={handleKeyDown}
            aria-label={t.chat.inputPlaceholder}
            placeholder={t.chat.inputPlaceholder}
            className="min-h-10 max-h-32 w-full resize-y rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-2 text-sm text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500 disabled:opacity-60"
          />
        </div>

        <button
          type="submit"
          disabled={!canSend}
          className="inline-flex h-10 shrink-0 items-center justify-center gap-1.5 rounded-md bg-neutral-900 px-4 text-sm font-medium text-white transition-colors duration-150 hover:bg-neutral-700 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-300 disabled:opacity-50"
        >
          <Send className="h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{t.chat.sendButton}</span>
        </button>
      </div>

      <p className="text-xs text-neutral-500 dark:text-neutral-400">
        {t.chat.photoHint}
      </p>
    </form>
  );
}
