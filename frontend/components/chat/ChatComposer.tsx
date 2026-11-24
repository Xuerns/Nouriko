"use client";

import {
  useEffect,
  useRef,
  useState,
  type ChangeEvent,
  type FormEvent,
  type KeyboardEvent,
} from "react";
import { Camera, ImagePlus, Loader2, Plus, Send, X } from "lucide-react";
import { CameraCaptureDialog } from "@/components/chat/CameraCaptureDialog";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Textarea } from "@/components/ui/textarea";
import { ApiError } from "@/lib/api";
import { ACCEPTED_PHOTO_INPUT, preparePhotoForUpload } from "@/lib/image";

interface ChatComposerProps {
  onSend: (payload: {
    message: string;
    photo: File | null;
    photoPreviewUrl?: string;
  }) => void;
  disabled?: boolean;
  onPhotoError?: (code: string | null) => void;
  prefillMessage?: string;
}

export function ChatComposer({
  onSend,
  disabled = false,
  onPhotoError,
  prefillMessage = "",
}: ChatComposerProps) {
  const { t } = useI18n();

  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const cameraInputRef = useRef<HTMLInputElement | null>(null);

  const [message, setMessage] = useState(prefillMessage);
  const [photoFile, setPhotoFile] = useState<File | null>(null);
  const [photoPreviewUrl, setPhotoPreviewUrl] = useState<string | null>(null);
  const [isPreparingPhoto, setIsPreparingPhoto] = useState(false);
  const [isCameraOpen, setIsCameraOpen] = useState(false);

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

  async function processSelectedPhoto(selected: File) {
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
      onPhotoError?.(null);
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

  async function handleFileChange(e: ChangeEvent<HTMLInputElement>) {
    const selected = e.target.files?.[0];
    e.target.value = "";
    if (!selected) return;
    await processSelectedPhoto(selected);
  }

  function handleTakePhotoClick() {
    if (
      typeof navigator !== "undefined" &&
      navigator.mediaDevices &&
      typeof navigator.mediaDevices.getUserMedia === "function"
    ) {
      setIsCameraOpen(true);
      return;
    }

    cameraInputRef.current?.click();
  }

  function handleAttachPhotoClick() {
    fileInputRef.current?.click();
  }

  function handleCameraCapture(file: File) {
    setIsCameraOpen(false);
    void processSelectedPhoto(file);
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
    <>
      <form
        onSubmit={handleSubmit}
        className="rounded-3xl border border-border bg-card p-3 space-y-3"
      >
        {photoPreviewUrl && photoFile ? (
          <div className="flex items-center justify-between gap-3 rounded-2xl bg-muted p-2.5">
            <div className="flex items-center gap-3 min-w-0">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={photoPreviewUrl}
                alt={photoFile.name}
                className="size-12 shrink-0 rounded-xl border border-border object-cover"
              />
              <div className="min-w-0">
                <p className="truncate text-xs font-medium text-foreground">
                  {photoFile.name}
                </p>
                <p className="text-xs tabular-nums text-muted-foreground">
                  {Math.max(1, Math.round(photoFile.size / 1024))} KB
                </p>
              </div>
            </div>

            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleRemovePhoto}
              disabled={disabled}
              aria-label={t.chat.removePhotoLabel}
              className="h-8 rounded-full px-3 text-xs shadow-none"
            >
              <X className="size-3.5 shrink-0 stroke-[1.75]" aria-hidden="true" />
              <span>{t.chat.removePhotoLabel}</span>
            </Button>
          </div>
        ) : null}

        {isPreparingPhoto ? (
          <div className="flex items-center gap-2 px-1 text-xs text-muted-foreground">
            <Loader2
              className="size-3.5 shrink-0 animate-spin stroke-[1.75]"
              aria-hidden="true"
            />
            <span>{t.chat.processingPhoto}</span>
          </div>
        ) : null}

        <div className="flex items-end gap-2">
          <input
            ref={fileInputRef}
            type="file"
            accept={ACCEPTED_PHOTO_INPUT}
            onChange={handleFileChange}
            className="sr-only"
            aria-label={t.chat.attachPhotoLabel}
          />
          <input
            ref={cameraInputRef}
            type="file"
            accept={ACCEPTED_PHOTO_INPUT}
            capture="environment"
            onChange={handleFileChange}
            className="sr-only"
            aria-label={t.chat.takePhotoLabel}
          />

          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  disabled={disabled || isPreparingPhoto}
                  aria-label={t.chat.photoMenuLabel}
                  title={t.chat.photoMenuLabel}
                  className="size-11 shrink-0 rounded-full border border-border shadow-none"
                />
              }
            >
              <Plus className="size-4 stroke-[1.75]" aria-hidden="true" />
            </DropdownMenuTrigger>
            <DropdownMenuContent
              side="top"
              align="start"
              sideOffset={8}
              className="w-48 rounded-2xl border border-border bg-card p-1.5 text-card-foreground shadow-none ring-0"
            >
              <DropdownMenuItem
                onClick={handleTakePhotoClick}
                className="rounded-xl px-3 py-2 text-xs font-medium"
              >
                <Camera className="size-4 stroke-[1.75]" aria-hidden="true" />
                <span>{t.chat.takePhotoLabel}</span>
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={handleAttachPhotoClick}
                className="rounded-xl px-3 py-2 text-xs font-medium"
              >
                <ImagePlus className="size-4 stroke-[1.75]" aria-hidden="true" />
                <span>{t.chat.attachPhotoLabel}</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          <Textarea
            rows={1}
            maxLength={1500}
            value={message}
            disabled={disabled}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={handleKeyDown}
            aria-label={t.chat.inputPlaceholder}
            placeholder={t.chat.inputPlaceholder}
            className="min-h-11 max-h-32 flex-1 resize-none border-0 bg-transparent px-2 py-2.5 text-sm text-foreground shadow-none placeholder:text-muted-foreground focus-visible:ring-0"
          />

          <Button
            type="submit"
            size="icon"
            disabled={!canSend}
            aria-label={t.chat.sendButton}
            title={t.chat.sendButton}
            className="size-11 shrink-0 rounded-full shadow-none"
          >
            <Send className="size-4 stroke-[1.75]" aria-hidden="true" />
          </Button>
        </div>
      </form>

      <CameraCaptureDialog
        open={isCameraOpen}
        onClose={() => setIsCameraOpen(false)}
        onCapture={handleCameraCapture}
        onFallbackToFile={() => fileInputRef.current?.click()}
      />
    </>
  );
}
