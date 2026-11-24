"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
} from "react";
import {
  Camera,
  CookingPot,
  ImagePlus,
  Loader2,
  ShoppingBasket,
  Utensils,
  Wallet,
  X,
} from "lucide-react";
import PromptBar, {
  type PromptBarCommand,
  type PromptBarSource,
} from "@/components/PromptBar";
import { CameraCaptureDialog } from "@/components/chat/CameraCaptureDialog";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api";
import { ACCEPTED_PHOTO_INPUT, preparePhotoForUpload } from "@/lib/image";

interface ChatComposerProps {
  onSend: (payload: {
    message: string;
    photo: File | null;
    photoPreviewUrl?: string;
  }) => void;
  onStop?: () => void;
  disabled?: boolean;
  onPhotoError?: (code: string | null) => void;
  prefillMessage?: string;
}

function commandIconFor(key: string) {
  const cls = "size-4 shrink-0 stroke-[1.75]";
  if (key === "budget") return <Wallet className={cls} aria-hidden="true" />;
  if (key === "grocery")
    return <ShoppingBasket className={cls} aria-hidden="true" />;
  if (key === "cookware")
    return <CookingPot className={cls} aria-hidden="true" />;
  return <Utensils className={cls} aria-hidden="true" />;
}

export function ChatComposer({
  onSend,
  onStop,
  disabled = false,
  onPhotoError,
  prefillMessage = "",
}: ChatComposerProps) {
  const { t } = useI18n();

  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const cameraInputRef = useRef<HTMLInputElement | null>(null);

  const [photoFile, setPhotoFile] = useState<File | null>(null);
  const [photoPreviewUrl, setPhotoPreviewUrl] = useState<string | null>(null);
  const [isPreparingPhoto, setIsPreparingPhoto] = useState(false);
  const [isCameraOpen, setIsCameraOpen] = useState(false);

  const mountedRef = useRef(true);
  const previewUrlRef = useRef<string | null>(null);
  const photoFileRef = useRef<File | null>(null);

  useEffect(() => {
    previewUrlRef.current = photoPreviewUrl;
  }, [photoPreviewUrl]);

  useEffect(() => {
    photoFileRef.current = photoFile;
  }, [photoFile]);

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

  const processSelectedPhoto = useCallback(
    async (selected: File) => {
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
    },
    [onPhotoError]
  );

  async function handleFileChange(e: ChangeEvent<HTMLInputElement>) {
    const selected = e.target.files?.[0];
    e.target.value = "";
    if (!selected) return;
    await processSelectedPhoto(selected);
  }

  const handleTakePhotoClick = useCallback(() => {
    if (
      typeof navigator !== "undefined" &&
      navigator.mediaDevices &&
      typeof navigator.mediaDevices.getUserMedia === "function"
    ) {
      setIsCameraOpen(true);
      return;
    }

    cameraInputRef.current?.click();
  }, []);

  const handleAttachPhotoClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleCameraCapture = useCallback(
    (file: File) => {
      setIsCameraOpen(false);
      void processSelectedPhoto(file);
    },
    [processSelectedPhoto]
  );

  const handleRemovePhoto = useCallback(() => {
    if (previewUrlRef.current) {
      URL.revokeObjectURL(previewUrlRef.current);
      previewUrlRef.current = null;
    }
    setPhotoFile(null);
    setPhotoPreviewUrl(null);
  }, []);

  const handleSourceAttach = useCallback(
    (sourceKey?: string) => {
      if (disabled || isPreparingPhoto) return;
      if (sourceKey === "take-photo") {
        handleTakePhotoClick();
        return;
      }
      handleAttachPhotoClick();
    },
    [disabled, handleAttachPhotoClick, handleTakePhotoClick, isPreparingPhoto]
  );

  const handlePromptSend = useCallback(
    (text: string) => {
      if (disabled || isPreparingPhoto) return;
      const trimmed = text.trim();
      const nextPhoto = photoFileRef.current;
      if (!trimmed && !nextPhoto) return;

      const nextPreview = previewUrlRef.current ?? undefined;
      previewUrlRef.current = null;
      photoFileRef.current = null;
      setPhotoFile(null);
      setPhotoPreviewUrl(null);

      onSend({
        message: trimmed,
        photo: nextPhoto,
        photoPreviewUrl: nextPreview,
      });
    },
    [disabled, isPreparingPhoto, onSend]
  );

  const sources = useMemo<PromptBarSource[]>(
    () => [
      {
        key: "take-photo",
        name: t.chat.takePhotoLabel,
        description: t.chat.takePhotoDescription,
        icon: <Camera className="size-4 shrink-0 stroke-[1.75]" aria-hidden="true" />,
        attach: true,
      },
      {
        key: "attach-photo",
        name: t.chat.attachPhotoLabel,
        description: t.chat.photoHint,
        icon: <ImagePlus className="size-4 shrink-0 stroke-[1.75]" aria-hidden="true" />,
        attach: true,
      },
    ],
    [
      t.chat.attachPhotoLabel,
      t.chat.photoHint,
      t.chat.takePhotoDescription,
      t.chat.takePhotoLabel,
    ]
  );

  const commands = useMemo<PromptBarCommand[]>(
    () =>
      t.chat.promptCommands.map((cmd) => ({
        key: cmd.key,
        name: cmd.name,
        description: cmd.description,
        insertText: cmd.insertText,
        icon: commandIconFor(cmd.key),
      })),
    [t.chat.promptCommands]
  );

  const labels = useMemo(
    () => ({
      promptAriaLabel: t.chat.inputPlaceholder,
      addAriaLabel: t.chat.photoMenuLabel,
      commandsButtonLabel: t.chat.promptTemplatesLabel,
      commandsAriaLabel: t.chat.promptTemplatesAriaLabel,
      sourcesAriaLabel: t.chat.photoMenuLabel,
      sendAriaLabel: t.chat.sendButton,
      stopAriaLabel: t.chat.stopButton,
      noMatchesText: t.chat.promptNoMatches,
    }),
    [
      t.chat.inputPlaceholder,
      t.chat.photoMenuLabel,
      t.chat.promptNoMatches,
      t.chat.promptTemplatesAriaLabel,
      t.chat.promptTemplatesLabel,
      t.chat.sendButton,
      t.chat.stopButton,
    ]
  );

  const attachmentSlot =
    (photoPreviewUrl && photoFile) || isPreparingPhoto ? (
      <div className="space-y-2">
        {photoPreviewUrl && photoFile ? (
          <div className="flex items-center justify-between gap-3 rounded-2xl bg-muted p-2.5">
            <div className="flex min-w-0 items-center gap-3">
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
              <X
                className="size-3.5 shrink-0 stroke-[1.75]"
                aria-hidden="true"
              />
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
      </div>
    ) : null;

  return (
    <>
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

      <PromptBar
        placeholder={t.chat.inputPlaceholder}
        initialValue={prefillMessage}
        maxLength={1500}
        disabled={disabled || isPreparingPhoto}
        busy={disabled}
        hasExternalAttachment={photoFile !== null}
        attachmentSlot={attachmentSlot}
        sources={sources}
        commands={commands}
        models={[]}
        showCommandsButton
        commandsButtonIcon={
          <svg
            viewBox="0 0 16 16"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.75"
            strokeLinecap="round"
            className="size-3.5 shrink-0"
            aria-hidden="true"
          >
            <path d="M9.75 2.5 6.25 13.5" />
          </svg>
        }
        labels={labels}
        onAttach={handleSourceAttach}
        onSend={handlePromptSend}
        onStop={onStop}
        background="var(--card)"
        color="var(--foreground)"
        menuBackground="var(--popover)"
        sendBackground="var(--brand-primary)"
        sendColor="var(--brand-primary-text)"
        width="100%"
        radius={24}
        maxRows={5}
      />

      <CameraCaptureDialog
        open={isCameraOpen}
        onClose={() => setIsCameraOpen(false)}
        onCapture={handleCameraCapture}
        onFallbackToFile={() => fileInputRef.current?.click()}
      />
    </>
  );
}
