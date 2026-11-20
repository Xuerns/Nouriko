import { ApiError } from "@/lib/api";

export const ACCEPTED_PHOTO_MIME_TYPES = [
  "image/jpeg",
  "image/png",
  "image/webp",
] as const;

export const ACCEPTED_PHOTO_INPUT = ACCEPTED_PHOTO_MIME_TYPES.join(",");
export const MAX_PHOTO_BYTES = 5 * 1024 * 1024; // 5 MiB
export const MAX_INPUT_PHOTO_BYTES = 25 * 1024 * 1024; // 25 MiB pre-decode guard
const MAX_DIMENSION_PX = 1280;
const PRIMARY_JPEG_QUALITY = 0.82;
const FALLBACK_JPEG_QUALITY = 0.68;

function isAcceptedMimeType(mime: string): boolean {
  return (ACCEPTED_PHOTO_MIME_TYPES as readonly string[]).includes(
    mime.toLowerCase()
  );
}

function loadImageElement(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new ApiError("PHOTO_INVALID", 400));
    img.src = src;
  });
}

function canvasToJpegBlob(
  canvas: HTMLCanvasElement,
  quality: number
): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          reject(new ApiError("PHOTO_INVALID", 400));
          return;
        }
        resolve(blob);
      },
      "image/jpeg",
      quality
    );
  });
}

function buildJpegFilename(originalName: string): string {
  const base = originalName.replace(/\.[^.]+$/, "").trim() || "photo";
  return `${base}.jpg`;
}

/**
 * Validates photo type (JPEG/PNG/WebP), resizes to max 1280px on the longest edge,
 * and compresses to JPEG (quality 0.82) via HTML5 Canvas so payload stays <= 5 MiB.
 */
export async function preparePhotoForUpload(file: File): Promise<File> {
  if (!file || !isAcceptedMimeType(file.type)) {
    throw new ApiError("PHOTO_INVALID", 400);
  }
  if (file.size === 0) {
    throw new ApiError("PHOTO_INVALID", 400);
  }
  if (file.size > MAX_PHOTO_BYTES) {
    throw new ApiError("PHOTO_TOO_LARGE", 413);
  }

  const objectUrl = URL.createObjectURL(file);
  try {
    const img = await loadImageElement(objectUrl);
    const width = img.naturalWidth || img.width;
    const height = img.naturalHeight || img.height;

    if (width <= 0 || height <= 0) {
      throw new ApiError("PHOTO_INVALID", 400);
    }

    const scale = Math.min(1, MAX_DIMENSION_PX / Math.max(width, height));
    const targetWidth = Math.max(1, Math.round(width * scale));
    const targetHeight = Math.max(1, Math.round(height * scale));

    const canvas = document.createElement("canvas");
    canvas.width = targetWidth;
    canvas.height = targetHeight;

    const ctx = canvas.getContext("2d");
    if (!ctx) {
      if (file.size <= MAX_PHOTO_BYTES) {
        return file;
      }
      throw new ApiError("PHOTO_INVALID", 400);
    }

    ctx.fillStyle = "#ffffff";
    ctx.fillRect(0, 0, targetWidth, targetHeight);
    ctx.drawImage(img, 0, 0, targetWidth, targetHeight);

    let blob = await canvasToJpegBlob(canvas, PRIMARY_JPEG_QUALITY);
    if (blob.size > MAX_PHOTO_BYTES) {
      blob = await canvasToJpegBlob(canvas, FALLBACK_JPEG_QUALITY);
    }
    if (blob.size === 0) {
      throw new ApiError("PHOTO_INVALID", 400);
    }
    if (blob.size > MAX_PHOTO_BYTES) {
      throw new ApiError("PHOTO_TOO_LARGE", 413);
    }

    return new File([blob], buildJpegFilename(file.name), {
      type: "image/jpeg",
      lastModified: Date.now(),
    });
  } finally {
    URL.revokeObjectURL(objectUrl);
  }
}
