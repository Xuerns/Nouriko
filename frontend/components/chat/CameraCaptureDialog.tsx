"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Camera, ImagePlus, Loader2, SwitchCamera } from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

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
            (err.name === "OverconstrainedError" ||
              err.name === "NotFoundError")
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

  function handleClose() {
    stopStream();
    setIsReady(false);
    setIsCapturing(false);
    setErrorMessage(null);
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
        handleClose();
        onCapture(file);
      },
      "image/jpeg",
      0.92
    );
  }

  function handleChooseExistingPhoto() {
    handleClose();
    onFallbackToFile();
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          handleClose();
        }
      }}
    >
      <DialogContent className="max-w-lg rounded-3xl border border-border bg-card p-5 shadow-none ring-0">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold text-foreground">
            {t.chat.cameraDialogTitle}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="relative aspect-4/3 w-full overflow-hidden rounded-2xl border border-border bg-muted">
            {errorMessage ? (
              <div className="flex h-full flex-col items-center justify-center gap-3 p-4 text-center">
                <p className="text-xs text-muted-foreground">{errorMessage}</p>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleChooseExistingPhoto}
                  className="h-9 rounded-full bg-card px-4 text-xs shadow-none"
                >
                  <ImagePlus
                    className="size-4 shrink-0 stroke-[1.75]"
                    aria-hidden="true"
                  />
                  <span>{t.chat.attachPhotoLabel}</span>
                </Button>
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
                  <div className="absolute inset-0 flex items-center justify-center gap-2 bg-muted text-xs text-muted-foreground">
                    <Loader2
                      className="size-4 shrink-0 animate-spin stroke-[1.75]"
                      aria-hidden="true"
                    />
                    <span>{t.chat.cameraStartingText}</span>
                  </div>
                ) : null}
              </>
            )}
          </div>

          <div className="flex flex-wrap items-center justify-between gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleSwitchCamera}
              disabled={Boolean(errorMessage) || isCapturing}
              className="h-10 rounded-full px-4 text-xs shadow-none"
            >
              <SwitchCamera
                className="size-4 shrink-0 stroke-[1.75]"
                aria-hidden="true"
              />
              <span>{t.chat.cameraSwitchButton}</span>
            </Button>

            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleClose}
                className="h-10 rounded-full px-4 text-xs shadow-none"
              >
                {t.chat.cameraCancelButton}
              </Button>
              <Button
                type="button"
                size="sm"
                onClick={handleCapture}
                disabled={!isReady || Boolean(errorMessage) || isCapturing}
                className="h-10 rounded-full px-4 text-xs shadow-none"
              >
                <Camera
                  className="size-4 shrink-0 stroke-[1.75]"
                  aria-hidden="true"
                />
                <span>{t.chat.cameraCaptureButton}</span>
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
