"use client";

import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type ChangeEvent,
  type FormEvent,
  type KeyboardEvent,
  type MouseEvent,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { Camera, ImagePlus, Loader2, Plus, Send, X } from "lucide-react";
import { CameraCaptureDialog } from "@/components/chat/CameraCaptureDialog";
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
  onPhotoError?: (code: string | null) => void;
}

type ExtendedFocusOptions = FocusOptions & { focusVisible?: boolean };

function focusElement(
  el: HTMLElement | null,
  options?: ExtendedFocusOptions
) {
  if (!el) return;
  try {
    el.focus(options);
  } catch {
    el.focus({ preventScroll: options?.preventScroll });
  }
}

export function ChatComposer({
  onSend,
  disabled = false,
  onPhotoError,
}: ChatComposerProps) {
  const { t } = useI18n();
  const triggerId = useId();
  const menuId = useId();

  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const cameraInputRef = useRef<HTMLInputElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const menuPopoverRef = useRef<HTMLDivElement | null>(null);
  const menuGroupRef = useRef<HTMLDivElement | null>(null);

  const pointerTriggerRef = useRef(false);
  const enterOnOpenRef = useRef(false);

  const [message, setMessage] = useState("");
  const [photoFile, setPhotoFile] = useState<File | null>(null);
  const [photoPreviewUrl, setPhotoPreviewUrl] = useState<string | null>(null);
  const [isPreparingPhoto, setIsPreparingPhoto] = useState(false);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
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

  const getEnabledMenuItems = useCallback((): HTMLButtonElement[] => {
    const group = menuGroupRef.current;
    if (!group) return [];
    return Array.from(group.querySelectorAll<HTMLButtonElement>("button")).filter(
      (item) =>
        !item.disabled && item.getAttribute("aria-disabled") !== "true"
    );
  }, []);

  const isPopoverOpen = useCallback((el: HTMLDivElement | null): boolean => {
    if (!el) return false;
    try {
      return el.matches(":popover-open");
    } catch {
      return el.classList.contains(":popover-open");
    }
  }, []);

  const closeMenuAndRestore = useCallback(
    (fromPointer: boolean) => {
      const popover = menuPopoverRef.current;
      const trigger = triggerRef.current;
      if (!popover || !isPopoverOpen(popover)) return;

      focusElement(trigger, {
        preventScroll: true,
        focusVisible: !fromPointer,
      });
      try {
        popover.hidePopover();
      } catch {
        // Ignore if popover is already hidden.
      }
    },
    [isPopoverOpen]
  );

  const positionMenuFallback = useCallback(() => {
    const popover = menuPopoverRef.current;
    const trigger = triggerRef.current;
    if (!popover || !trigger) return;
    if (
      typeof CSS !== "undefined" &&
      typeof CSS.supports === "function" &&
      CSS.supports("position-area", "block-start")
    ) {
      return;
    }

    const triggerRect = trigger.getBoundingClientRect();
    const menuRect = popover.getBoundingClientRect();
    const gap = 8;
    const viewportPadding = 8;

    let top = triggerRect.top - menuRect.height - gap;
    if (top < viewportPadding) {
      top = Math.min(
        window.innerHeight - menuRect.height - viewportPadding,
        triggerRect.bottom + gap
      );
    }

    const left = Math.max(
      viewportPadding,
      Math.min(
        triggerRect.left,
        window.innerWidth - menuRect.width - viewportPadding
      )
    );

    popover.style.left = `${Math.round(left)}px`;
    popover.style.top = `${Math.round(top)}px`;
  }, []);

  // Fallback roles & arrow-key navigation when native focusgroup is unsupported.
  useEffect(() => {
    const group = menuGroupRef.current;
    if (!group || "focusGroup" in HTMLElement.prototype) return;

    group.setAttribute("role", "menu");
    group.querySelectorAll("button:not([role])").forEach((item) => {
      item.setAttribute("role", "menuitem");
    });

    const handleGroupKeyDown = (event: globalThis.KeyboardEvent) => {
      if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
      const items = getEnabledMenuItems();
      if (items.length === 0) return;

      const active = document.activeElement as HTMLButtonElement | null;
      const current = active ? items.indexOf(active) : -1;
      if (current < 0) return;

      const next =
        event.key === "Home"
          ? 0
          : event.key === "End"
            ? items.length - 1
            : (current + (event.key === "ArrowUp" ? -1 : 1) + items.length) %
              items.length;

      event.preventDefault();
      event.stopImmediatePropagation();
      items[next]?.focus({ preventScroll: true });
    };

    group.addEventListener("keydown", handleGroupKeyDown, true);
    return () => {
      group.removeEventListener("keydown", handleGroupKeyDown, true);
    };
  }, [getEnabledMenuItems]);

  // Popover toggle lifecycle, fallback positioning, Escape/Tab, and outside focus handling.
  useEffect(() => {
    const popover = menuPopoverRef.current;
    const trigger = triggerRef.current;
    if (!popover || !trigger) return;

    const handleToggle = (event: Event) => {
      const toggleEvent = event as ToggleEvent;
      const open = toggleEvent.newState === "open";
      setIsMenuOpen(open);

      if (open) {
        positionMenuFallback();
        if (enterOnOpenRef.current) {
          enterOnOpenRef.current = false;
          getEnabledMenuItems()[0]?.focus({ preventScroll: true });
        } else if (pointerTriggerRef.current) {
          pointerTriggerRef.current = false;
          focusElement(trigger, { preventScroll: true, focusVisible: false });
        }
      } else {
        enterOnOpenRef.current = false;
        pointerTriggerRef.current = false;
      }
    };

    const handleMenuKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        closeMenuAndRestore(false);
        return;
      }
      if (event.key === "Tab" && isPopoverOpen(popover)) {
        try {
          popover.hidePopover();
        } catch {
          // Ignore if already closed.
        }
      }
    };

    const handleDocumentFocusIn = (event: FocusEvent) => {
      if (!isPopoverOpen(popover)) return;
      const target = event.target as Node | null;
      if (target && !popover.contains(target) && target !== trigger) {
        try {
          popover.hidePopover();
        } catch {
          // Ignore if already closed.
        }
      }
    };

    popover.addEventListener("toggle", handleToggle);
    popover.addEventListener("keydown", handleMenuKeyDown);
    document.addEventListener("focusin", handleDocumentFocusIn);

    return () => {
      popover.removeEventListener("toggle", handleToggle);
      popover.removeEventListener("keydown", handleMenuKeyDown);
      document.removeEventListener("focusin", handleDocumentFocusIn);
    };
  }, [
    closeMenuAndRestore,
    getEnabledMenuItems,
    isPopoverOpen,
    positionMenuFallback,
  ]);

  function handleTriggerPointerDown(e: ReactPointerEvent<HTMLButtonElement>) {
    e.preventDefault();
    pointerTriggerRef.current = true;
  }

  function handleTriggerClick(e: MouseEvent<HTMLButtonElement>) {
    const popover = menuPopoverRef.current;
    const trigger = triggerRef.current;
    if (!popover || !trigger) return;

    const fromKeyboard = e.detail === 0;
    if (fromKeyboard) {
      pointerTriggerRef.current = false;
    }

    if ("commandForElement" in HTMLButtonElement.prototype) {
      if (fromKeyboard && !isPopoverOpen(popover)) {
        enterOnOpenRef.current = true;
      }
      return;
    }

    // Fallback toggle when native Invoker Commands are unsupported.
    e.preventDefault();
    if (isPopoverOpen(popover)) {
      closeMenuAndRestore(!fromKeyboard);
      return;
    }

    enterOnOpenRef.current = fromKeyboard;
    try {
      (
        popover as HTMLDivElement & {
          showPopover: (options?: { source?: HTMLElement }) => void;
        }
      ).showPopover({ source: trigger });
    } catch {
      popover.showPopover();
    }
  }

  function handleTriggerKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
    const popover = menuPopoverRef.current;
    const trigger = triggerRef.current;
    if (!popover || !trigger) return;

    e.preventDefault();
    if (isPopoverOpen(popover)) {
      const items = getEnabledMenuItems();
      const target =
        e.key === "ArrowUp" ? items[items.length - 1] : items[0];
      target?.focus({ preventScroll: true });
      return;
    }

    enterOnOpenRef.current = true;
    pointerTriggerRef.current = false;
    trigger.click();
  }

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

  function handleTakePhotoClick(e: MouseEvent<HTMLButtonElement>) {
    const fromPointer = e.detail !== 0;
    closeMenuAndRestore(fromPointer);

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

  function handleAttachPhotoClick(e: MouseEvent<HTMLButtonElement>) {
    const fromPointer = e.detail !== 0;
    closeMenuAndRestore(fromPointer);
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
    <>
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
            <input
              ref={cameraInputRef}
              type="file"
              accept={ACCEPTED_PHOTO_INPUT}
              capture="environment"
              onChange={handleFileChange}
              className="sr-only"
              aria-label={t.chat.takePhotoLabel}
            />

            <button
              ref={triggerRef}
              id={triggerId}
              type="button"
              disabled={disabled || isPreparingPhoto}
              aria-haspopup="menu"
              aria-expanded={isMenuOpen}
              aria-label={t.chat.photoMenuLabel}
              title={t.chat.photoMenuLabel}
              onPointerDown={handleTriggerPointerDown}
              onClick={handleTriggerClick}
              onKeyDown={handleTriggerKeyDown}
              {...({
                commandfor: menuId,
                command: "toggle-popover",
              } as Record<string, unknown>)}
              className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md border border-neutral-300 dark:border-neutral-700 text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 aria-expanded:bg-neutral-100 dark:aria-expanded:bg-neutral-900 disabled:opacity-50"
            >
              <Plus className="h-4 w-4 shrink-0" aria-hidden="true" />
            </button>

            <div
              ref={menuPopoverRef}
              id={menuId}
              popover="auto"
              className="composer-photo-menu min-w-44 rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 p-1 text-neutral-900 dark:text-neutral-100"
            >
              <div
                ref={menuGroupRef}
                aria-labelledby={triggerId}
                {...({ focusgroup: "menu nomemory" } as Record<
                  string,
                  unknown
                >)}
                className="flex flex-col gap-0.5"
              >
                <button
                  type="button"
                  onClick={handleTakePhotoClick}
                  className="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 focus-visible:bg-neutral-100 dark:focus-visible:bg-neutral-900 focus:outline-none"
                >
                  <Camera className="h-4 w-4 shrink-0" aria-hidden="true" />
                  <span>{t.chat.takePhotoLabel}</span>
                </button>
                <button
                  type="button"
                  onClick={handleAttachPhotoClick}
                  className="flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 focus-visible:bg-neutral-100 dark:focus-visible:bg-neutral-900 focus:outline-none"
                >
                  <ImagePlus className="h-4 w-4 shrink-0" aria-hidden="true" />
                  <span>{t.chat.attachPhotoLabel}</span>
                </button>
              </div>
            </div>

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
