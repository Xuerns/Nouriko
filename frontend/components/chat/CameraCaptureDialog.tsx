"use client";

import { useCallback, useEffect, useId, useRef, useState } from "react";
import { Camera, ImagePlus, Loader2, SwitchCamera, X } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";

type FacingMode = "environment" | "user";

interface CameraCaptureDialogProps {
  open: boolean;
  onClose: () => void;
  onCapture: (file: File) => void;
  onFallbackToFile: () => void;
}

export function CameraCaptureDialog({
  open,
  onClose,
  onCapture,
  onFallbackToFile,
}: CameraCaptureDialogProps) {
  const { t } = useI18n();
  const titleId = useId();
  const dialogRef = useRef<HTMLDialogElement | null>(null);
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const streamRef = useRef<MediaStream | null>(null);

  const [facingMode, setFacingMode] = useState<FacingMode>("environment");
  const [isReady, setIsReady] = useState(false);
  const [isCapturing, setIsCapturing] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const stopStream = useCallback(() => {
    const stream = streamRef.current;
    if (stream) {
      for (const track of stream.getTracks()) {
        track.stop();
      }
      streamRef.current = null;
    }
    if (videoRef.current) {
      videoRef.current.srcObject = null;
    }
  }, []);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open) {
      if (!dialog.open) {
        dialog.showModal();
      }
    } else if (dialog.open) {
      dialog.close();
    }
  }, [open]);

  // Fallback light-dismiss for browsers without native <dialog closedby="any">.
  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog || "closedBy" in HTMLDialogElement.prototype) return;

    const isOutside = (e: PointerEvent) => {
      const rect = dialog.getBoundingClientRect();
      return (
        e.clientX < rect.left ||
        e.clientX > rect.right ||
        e.clientY < rect.top ||
        e.clientY > rect.bottom
      );
    };

    let pressedOutside = false;
    const handlePointerDown = (e: PointerEvent) => {
      pressedOutside = isOutside(e);
    };
    const handlePointerUp = (e: PointerEvent) => {
      if (pressedOutside && isOutside(e) && dialog.open) {
        dialog.close();
      }
      pressedOutside = false;
    };

    dialog.addEventListener("pointerdown", handlePointerDown);
    dialog.addEventListener("pointerup", handlePointerUp);
    return () => {
      dialog.removeEventListener("pointerdown", handlePointerDown);
      dialog.removeEventListener("pointerup", handlePointerUp);
    };
  }, []);

  useEffect(() => {
    if (!open) {
      stopStream();
      return;
    }

    let cancelled = false;

    async function startCamera() {
      if (
        typeof navigator === "undefined" ||
        !navigator.mediaDevices ||
        typeof navigator.mediaDevices.getUserMedia !== "function"
      ) {
        if (!cancelled) {
          setErrorMessage(t.chat.cameraUnavailableError);
        }
        return;
      }

      try {
        let stream: MediaStream;
        try {
          stream = await navigator.mediaDevices.getUserMedia({
            video: { facingMode: { ideal: facingMode } },
            audio: false,
          });
        } catch (err) {
          if (
            err instanceof DOMException &&
            (err.name === "OverconstrainedError" || err.name === "NotFoundError")
          ) {
            stream = await navigator.mediaDevices.getUserMedia({
              video: true,
              audio: false,
            });
          } else {
            throw err;
          }
        }

        if (cancelled) {
          for (const track of stream.getTracks()) {
            track.stop();
          }
          return;
        }

        streamRef.current = stream;
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
        }
      } catch {
        if (!cancelled) {
          setErrorMessage(t.chat.cameraPermissionError);
        }
      }
    }

    void startCamera();

    return () => {
      cancelled = true;
      stopStream();
    };
  }, [
    open,
    facingMode,
    stopStream,
    t.chat.cameraPermissionError,
    t.chat.cameraUnavailableError,
  ]);

  function handleDialogClose() {
    stopStream();
    setIsReady(false);
    setIsCapturing(false);
    setErrorMessage(null);
    if (dialogRef.current) {
      dialogRef.current.returnValue = "";
    }
    onClose();
  }

  function handleSwitchCamera() {
    stopStream();
    setIsReady(false);
    setErrorMessage(null);
    setFacingMode((prev) => (prev === "environment" ? "user" : "environment"));
  }

  function handleCapture() {
    const video = videoRef.current;
    if (!video || !isReady || isCapturing) return;

    const width = video.videoWidth;
    const height = video.videoHeight;
    if (width <= 0 || height <= 0) return;

    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;

    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    setIsCapturing(true);
    ctx.drawImage(video, 0, 0, width, height);
    canvas.toBlob(
      (blob) => {
        setIsCapturing(false);
        if (!blob) return;

        const file = new File([blob], `camera-${Date.now()}.jpg`, {
          type: "image/jpeg",
          lastModified: Date.now(),
        });
        dialogRef.current?.close();
        onCapture(file);
      },
      "image/jpeg",
      0.92
    );
  }

  function handleChooseExistingPhoto() {
    dialogRef.current?.close();
    onFallbackToFile();
  }

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleId}
      onClose={handleDialogClose}
      {...({ closedby: "any" } as Record<string, unknown>)}
      className="m-auto w-[calc(100%-2rem)] max-w-lg rounded-lg border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 p-0 text-neutral-900 dark:text-neutral-100 backdrop:bg-neutral-950/60"
    >
      <div className="flex items-center justify-between border-b border-neutral-200 dark:border-neutral-800 px-4 py-3">
        <h2
          id={titleId}
          className="text-sm font-semibold text-neutral-900 dark:text-neutral-100"
        >
          {t.chat.cameraDialogTitle}
        </h2>
        <button
          type="button"
          onClick={() => dialogRef.current?.close()}
          aria-label={t.chat.cameraCancelButton}
          className="inline-flex h-8 w-8 items-center justify-center rounded-md border border-neutral-300 dark:border-neutral-700 text-neutral-700 dark:text-neutral-300 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
        >
          <X className="h-4 w-4 shrink-0" aria-hidden="true" />
        </button>
      </div>

      <div className="p-4 space-y-4">
        <div className="relative aspect-4/3 w-full overflow-hidden rounded-md border border-neutral-200 dark:border-neutral-800 bg-neutral-100 dark:bg-neutral-900">
          {errorMessage ? (
            <div className="flex h-full flex-col items-center justify-center gap-3 p-4 text-center">
              <p className="text-xs text-neutral-700 dark:text-neutral-300">
                {errorMessage}
              </p>
              <button
                type="button"
                onClick={handleChooseExistingPhoto}
                className="inline-flex h-9 items-center justify-center gap-1.5 rounded-md border border-neutral-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 px-3 text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
              >
                <ImagePlus className="h-4 w-4 shrink-0" aria-hidden="true" />
                <span>{t.chat.attachPhotoLabel}</span>
              </button>
            </div>
          ) : (
            <>
              <video
                ref={videoRef}
                autoPlay
                playsInline
                muted
                onLoadedMetadata={() => setIsReady(true)}
                className="h-full w-full object-cover"
              />
              {!isReady ? (
                <div className="absolute inset-0 flex items-center justify-center gap-2 bg-neutral-100 dark:bg-neutral-900 text-xs text-neutral-600 dark:text-neutral-400">
                  <Loader2
                    className="h-4 w-4 shrink-0 animate-spin"
                    aria-hidden="true"
                  />
                  <span>{t.chat.cameraStartingText}</span>
                </div>
              ) : null}
            </>
          )}
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2">
          <button
            type="button"
            onClick={handleSwitchCamera}
            disabled={Boolean(errorMessage) || isCapturing}
            className="inline-flex h-9 items-center justify-center gap-1.5 rounded-md border border-neutral-300 dark:border-neutral-700 px-3 text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
          >
            <SwitchCamera className="h-4 w-4 shrink-0" aria-hidden="true" />
            <span>{t.chat.cameraSwitchButton}</span>
          </button>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => dialogRef.current?.close()}
              className="inline-flex h-9 items-center justify-center rounded-md border border-neutral-300 dark:border-neutral-700 px-3 text-xs font-medium text-neutral-800 dark:text-neutral-200 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900"
            >
              {t.chat.cameraCancelButton}
            </button>
            <button
              type="button"
              autoFocus
              onClick={handleCapture}
              disabled={!isReady || Boolean(errorMessage) || isCapturing}
              className="inline-flex h-9 items-center justify-center gap-1.5 rounded-md bg-neutral-900 px-3.5 text-xs font-medium text-white transition-colors duration-150 hover:bg-neutral-700 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-300 disabled:opacity-50"
            >
              <Camera className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{t.chat.cameraCaptureButton}</span>
            </button>
          </div>
        </div>
      </div>
    </dialog>
  );
}
