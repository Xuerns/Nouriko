"use client";

import { useId, useState, type FormEvent } from "react";
import { useI18n } from "@/components/providers/I18nProvider";
import { Button } from "@/components/ui/button";
import { FieldError } from "@/components/ui/FieldError";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { focusFieldById } from "@/lib/form-validation";
import type { ChatIntent } from "@/lib/types";
import { cn } from "@/lib/utils";

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
  const baseId = useId();
  const nameInputId = `${baseId}-food-name`;
  const portionInputId = `${baseId}-portion`;
  const nameErrorId = `${baseId}-food-name-error`;

  const [expanded, setExpanded] = useState(needsClarification);
  const [photoType, setPhotoType] = useState<
    "photo_food" | "photo_ingredients"
  >(intent === "photo_ingredients" ? "photo_ingredients" : "photo_food");
  const [name, setName] = useState(
    detectedItems.length > 0 ? detectedItems.join(", ") : defaultFoodName
  );
  const [portion, setPortion] = useState("");
  const [nameError, setNameError] = useState<string | null>(null);

  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (disabled) return;

    const trimmedName = name.trim();
    if (!trimmedName) {
      setNameError(t.common.foodNameRequired);
      focusFieldById(nameInputId);
      return;
    }

    setNameError(null);
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
    <div className="rounded-2xl bg-muted p-4 space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="space-y-0.5">
          <p className="text-xs font-medium text-foreground">
            {expanded
              ? t.chat.photoCorrectionHeading
              : t.chat.photoCorrectionPrompt}
          </p>
          {detectedItems.length > 0 ? (
            <p className="text-xs text-muted-foreground">
              {t.chat.detectedItemsLabel}: {detectedItems.join(", ")}
            </p>
          ) : null}
        </div>

        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => {
            setNameError(null);
            setExpanded((prev) => !prev);
          }}
          disabled={disabled}
          className="h-9 rounded-full bg-card px-3.5 text-xs shadow-none"
        >
          {expanded ? t.common.cancel : t.chat.photoCorrectionToggle}
        </Button>
      </div>

      {expanded ? (
        <>
          <Separator className="bg-border" />
          <form noValidate onSubmit={handleSubmit} className="space-y-3">
            <div className="space-y-1.5">
              <span className="block text-xs font-medium text-foreground">
                {t.chat.photoTypeLabel}
              </span>
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant={photoType === "photo_food" ? "default" : "outline"}
                  onClick={() => setPhotoType("photo_food")}
                  className="h-9 rounded-full px-4 text-xs shadow-none"
                >
                  {t.chat.photoTypeFood}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant={
                    photoType === "photo_ingredients" ? "default" : "outline"
                  }
                  onClick={() => setPhotoType("photo_ingredients")}
                  className="h-9 rounded-full px-4 text-xs shadow-none"
                >
                  {t.chat.photoTypeIngredients}
                </Button>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label
                  htmlFor={nameInputId}
                  className="text-xs font-medium text-foreground"
                >
                  {t.chat.photoNameLabel}
                </Label>
                <Input
                  id={nameInputId}
                  name="food_name"
                  type="text"
                  required
                  maxLength={200}
                  value={name}
                  onChange={(e) => {
                    const next = e.target.value;
                    setName(next);
                    if (nameError && next.trim()) {
                      setNameError(null);
                    }
                  }}
                  onBlur={() => {
                    if (!name.trim()) {
                      setNameError(t.common.foodNameRequired);
                    }
                  }}
                  aria-invalid={Boolean(nameError)}
                  aria-describedby={nameError ? nameErrorId : undefined}
                  placeholder={t.chat.photoNamePlaceholder}
                  className={cn(
                    "h-10 rounded-2xl bg-card px-3 text-sm shadow-none",
                    nameError &&
                      "border-[var(--error-text)] focus-visible:border-[var(--error-text)]"
                  )}
                />
                <FieldError id={nameErrorId} message={nameError} />
              </div>

              <div className="space-y-1.5">
                <Label
                  htmlFor={portionInputId}
                  className="text-xs font-medium text-foreground"
                >
                  {t.chat.photoPortionLabel}
                </Label>
                <Input
                  id={portionInputId}
                  name="portion"
                  type="text"
                  maxLength={100}
                  value={portion}
                  onChange={(e) => setPortion(e.target.value)}
                  placeholder={t.chat.photoPortionPlaceholder}
                  className="h-10 rounded-2xl bg-card px-3 text-sm shadow-none"
                />
              </div>
            </div>

            <div className="flex justify-end">
              <Button
                type="submit"
                size="sm"
                disabled={disabled}
                className="h-9 rounded-full px-4 text-xs shadow-none"
              >
                {t.chat.photoCorrectionSubmit}
              </Button>
            </div>
          </form>
        </>
      ) : null}
    </div>
  );
}
