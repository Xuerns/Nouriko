"use client";

import { useState, type FormEvent } from "react";
import { useI18n } from "@/components/providers/I18nProvider";
import type { ChatIntent } from "@/lib/types";

interface PhotoCorrectionBarProps {
  intent: ChatIntent;
  detectedItems: string[];
  needsClarification: boolean;
  defaultFoodName?: string;
  disabled?: boolean;
  onSubmitCorrection: (message: string) => void;
}

export function PhotoCorrectionBar({
  intent,
  detectedItems,
  needsClarification,
  defaultFoodName = "",
  disabled = false,
  onSubmitCorrection,
}: PhotoCorrectionBarProps) {
  const { t, language } = useI18n();
  const [expanded, setExpanded] = useState(needsClarification);
  const [photoType, setPhotoType] = useState<
    "photo_food" | "photo_ingredients"
  >(intent === "photo_ingredients" ? "photo_ingredients" : "photo_food");
  const [name, setName] = useState(
    detectedItems.length > 0 ? detectedItems.join(", ") : defaultFoodName
  );
  const [portion, setPortion] = useState("");

  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName || disabled) return;

    const trimmedPortion = portion.trim();
    let followUp: string;
    if (language === "en") {
      if (photoType === "photo_ingredients") {
        followUp = trimmedPortion
          ? `Correction for the previous photo: these are raw ingredients (${trimmedName}, ${trimmedPortion}). Please suggest healthy meals I can cook with them.`
          : `Correction for the previous photo: these are raw ingredients (${trimmedName}). Please suggest healthy meals I can cook with them.`;
      } else {
        followUp = trimmedPortion
          ? `Correction for the previous photo: this is a prepared meal (${trimmedName}, portion: ${trimmedPortion}). Please recalculate the nutrition estimate.`
          : `Correction for the previous photo: this is a prepared meal (${trimmedName}). Please recalculate the nutrition estimate.`;
      }
    } else {
      if (photoType === "photo_ingredients") {
        followUp = trimmedPortion
          ? `Koreksi foto sebelumnya: ini bahan mentah (${trimmedName}, ${trimmedPortion}). Tolong rekomendasikan menu sehat yang bisa dimasak.`
          : `Koreksi foto sebelumnya: ini bahan mentah (${trimmedName}). Tolong rekomendasikan menu sehat yang bisa dimasak.`;
      } else {
        followUp = trimmedPortion
          ? `Koreksi foto sebelumnya: ini makanan jadi (${trimmedName}, porsi: ${trimmedPortion}). Tolong hitung ulang perkiraan nutrisinya.`
          : `Koreksi foto sebelumnya: ini makanan jadi (${trimmedName}). Tolong hitung ulang perkiraan nutrisinya.`;
      }
    }

    onSubmitCorrection(followUp);
    setExpanded(false);
  }

  return (
    <div className="rounded-md border border-neutral-300 dark:border-neutral-700 p-3 space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="space-y-0.5">
          <p className="text-xs font-medium text-neutral-800 dark:text-neutral-200">
            {expanded
              ? t.chat.photoCorrectionHeading
              : t.chat.photoCorrectionPrompt}
          </p>
          {detectedItems.length > 0 ? (
            <p className="text-xs text-neutral-500 dark:text-neutral-400">
              {t.chat.detectedItemsLabel}: {detectedItems.join(", ")}
            </p>
          ) : null}
        </div>

        <button
          type="button"
          onClick={() => setExpanded((prev) => !prev)}
          disabled={disabled}
          className="rounded-md border border-neutral-300 dark:border-neutral-700 px-2.5 py-1 text-xs font-medium text-neutral-900 dark:text-neutral-100 transition-colors duration-150 hover:bg-neutral-100 dark:hover:bg-neutral-900 disabled:opacity-50"
        >
          {expanded ? t.common.cancel : t.chat.photoCorrectionToggle}
        </button>
      </div>

      {expanded ? (
        <form onSubmit={handleSubmit} className="space-y-3 border-t border-neutral-300 dark:border-neutral-700 pt-3">
          <div className="space-y-1.5">
            <span className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
              {t.chat.photoTypeLabel}
            </span>
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => setPhotoType("photo_food")}
                className={`rounded-md border px-3 py-1.5 text-xs font-medium transition-colors duration-150 ${
                  photoType === "photo_food"
                    ? "border-neutral-900 bg-neutral-900 text-white dark:border-neutral-100 dark:bg-neutral-100 dark:text-neutral-950"
                    : "border-neutral-300 dark:border-neutral-700 text-neutral-800 dark:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-900"
                }`}
              >
                {t.chat.photoTypeFood}
              </button>
              <button
                type="button"
                onClick={() => setPhotoType("photo_ingredients")}
                className={`rounded-md border px-3 py-1.5 text-xs font-medium transition-colors duration-150 ${
                  photoType === "photo_ingredients"
                    ? "border-neutral-900 bg-neutral-900 text-white dark:border-neutral-100 dark:bg-neutral-100 dark:text-neutral-950"
                    : "border-neutral-300 dark:border-neutral-700 text-neutral-800 dark:text-neutral-200 hover:bg-neutral-100 dark:hover:bg-neutral-900"
                }`}
              >
                {t.chat.photoTypeIngredients}
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div className="space-y-1">
              <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                {t.chat.photoNameLabel}
                <input
                  type="text"
                  required
                  maxLength={200}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder={t.chat.photoNamePlaceholder}
                  className="mt-1 w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-1.5 text-sm font-normal text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500"
                />
              </label>
            </div>

            <div className="space-y-1">
              <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                {t.chat.photoPortionLabel}
                <input
                  type="text"
                  maxLength={100}
                  value={portion}
                  onChange={(e) => setPortion(e.target.value)}
                  placeholder={t.chat.photoPortionPlaceholder}
                  className="mt-1 w-full rounded-md border border-neutral-300 dark:border-neutral-700 bg-transparent px-3 py-1.5 text-sm font-normal text-neutral-900 dark:text-neutral-100 placeholder:text-neutral-500"
                />
              </label>
            </div>
          </div>

          <div className="flex justify-end">
            <button
              type="submit"
              disabled={disabled || !name.trim()}
              className="rounded-md bg-neutral-900 px-3.5 py-1.5 text-xs font-medium text-white transition-colors duration-150 hover:bg-neutral-700 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-neutral-300 disabled:opacity-50"
            >
              {t.chat.photoCorrectionSubmit}
            </button>
          </div>
        </form>
      ) : null}
    </div>
  );
}
