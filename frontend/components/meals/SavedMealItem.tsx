"use client";

import { useState } from "react";
import { Dialog as DialogPrimitive } from "@base-ui/react/dialog";
import {
  Beef,
  ChevronRight,
  Cookie,
  Droplet,
  Flame,
  Leaf,
  Loader2,
  Moon,
  Sun,
  Sunrise,
  Trash2,
  Wallet,
  Wheat,
  X,
} from "lucide-react";
import { useI18n } from "@/components/providers/I18nProvider";
import { formatIDR, formatMacro } from "@/components/chat/format";
import { getHealthBadgeClass } from "@/components/chat/MealResultCard";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Dialog,
  DialogClose,
  DialogDescription,
  DialogHeader,
  DialogOverlay,
  DialogPortal,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { SavedMealDTO } from "@/lib/types";

interface SavedMealItemProps {
  meal: SavedMealDTO;
  onDelete?: (mealId: string) => void | Promise<void>;
  isDeleting?: boolean;
  surface?: "card" | "canvas";
  desktopLayout?: "compact" | "expanded";
}

const MEAL_TIME_ICONS = {
  breakfast: Sunrise,
  lunch: Sun,
  dinner: Moon,
  snack: Cookie,
} as const;

interface DeleteIconActionProps {
  mealName: string;
  isDeleting: boolean;
  deleteLabel: string;
  deletingLabel: string;
  cancelLabel: string;
  buttonClassName: string;
  onConfirm: () => void | Promise<void>;
}

function DeleteIconAction({
  mealName,
  isDeleting,
  deleteLabel,
  deletingLabel,
  cancelLabel,
  buttonClassName,
  onConfirm,
}: DeleteIconActionProps) {
  return (
    <AlertDialog>
      <Tooltip>
        <TooltipTrigger
          render={
            <AlertDialogTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  disabled={isDeleting}
                  aria-label={`${deleteLabel} ${mealName}`}
                  className={buttonClassName}
                />
              }
            />
          }
        >
          {isDeleting ? (
            <Loader2
              className="size-4 shrink-0 animate-spin stroke-[1.75]"
              aria-hidden="true"
            />
          ) : (
            <Trash2 className="size-4 shrink-0 stroke-[1.75]" aria-hidden="true" />
          )}
        </TooltipTrigger>
        <TooltipContent className="rounded-xl shadow-none">
          {isDeleting ? deletingLabel : deleteLabel}
        </TooltipContent>
      </Tooltip>
      <AlertDialogContent className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0">
        <AlertDialogHeader>
          <AlertDialogTitle>{deleteLabel}</AlertDialogTitle>
          <AlertDialogDescription>{mealName}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter className="mx-0 mb-0 rounded-none border-0 bg-transparent p-0 pt-2">
          <AlertDialogCancel className="h-11 rounded-full px-5 shadow-none">
            {cancelLabel}
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => void onConfirm()}
            className="h-11 rounded-full bg-[var(--error-bg)] px-5 text-[var(--error-text)] hover:bg-[var(--error-bg)]/80 shadow-none"
          >
            {deleteLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

export function SavedMealItem({
  meal,
  onDelete,
  isDeleting = false,
  surface = "card",
  desktopLayout = "compact",
}: SavedMealItemProps) {
  const { t } = useI18n();
  const [detailOpen, setDetailOpen] = useState(false);

  const mealTimeLabel = Object.hasOwn(t.meals.mealTimes, meal.meal_time)
    ? t.meals.mealTimes[meal.meal_time]
    : meal.meal_time;

  const sourceLabel = Object.hasOwn(t.meals.inputSources, meal.input_source)
    ? t.meals.inputSources[meal.input_source]
    : meal.input_source;

  const healthLabelText = Object.hasOwn(t.chat.healthLabels, meal.health_label)
    ? t.chat.healthLabels[meal.health_label]
    : t.chat.healthLabels.fair;

  const MealTimeIcon = Object.hasOwn(MEAL_TIME_ICONS, meal.meal_time)
    ? MEAL_TIME_ICONS[meal.meal_time as keyof typeof MEAL_TIME_ICONS]
    : Sun;

  const macros = [
    {
      label: t.chat.caloriesLabel,
      value: `${Math.round(meal.calories)}`,
      unit: t.profile.kcalUnit,
      icon: Flame,
      colorVar: "var(--macro-calories)",
    },
    {
      label: t.chat.proteinLabel,
      value: formatMacro(meal.protein_g),
      unit: t.profile.gramUnit,
      icon: Beef,
      colorVar: "var(--macro-protein)",
    },
    {
      label: t.chat.carbsLabel,
      value: formatMacro(meal.carbs_g),
      unit: t.profile.gramUnit,
      icon: Wheat,
      colorVar: "var(--macro-carbs)",
    },
    {
      label: t.chat.fatLabel,
      value: formatMacro(meal.fat_g),
      unit: t.profile.gramUnit,
      icon: Droplet,
      colorVar: "var(--macro-fat)",
    },
    {
      label: t.chat.fiberLabel,
      value: formatMacro(meal.fiber_g),
      unit: t.profile.gramUnit,
      icon: Leaf,
      colorVar: "var(--macro-fiber)",
    },
  ];

  const hasPrice = meal.price_idr !== null && meal.price_idr > 0;
  const showExpandedOnDesktop = desktopLayout === "expanded";

  async function handleConfirmDelete() {
    if (!onDelete) return;
    setDetailOpen(false);
    await onDelete(meal.id);
  }

  return (
    <>
      {showExpandedOnDesktop ? (
        <Card
          className={`hidden flex-col justify-between gap-4 rounded-2xl border border-border p-5 shadow-none ring-0 lg:flex ${
            surface === "canvas" ? "bg-background" : "rounded-3xl bg-card"
          }`}
        >
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 space-y-1">
              <h3 className="text-base font-semibold leading-snug text-foreground">
                {meal.name}
              </h3>
              <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                <span className="inline-flex items-center gap-1.5">
                  <MealTimeIcon
                    className="size-3.5 shrink-0 stroke-[1.75]"
                    aria-hidden="true"
                  />
                  <span>
                    {mealTimeLabel} · {sourceLabel}
                  </span>
                </span>
                {hasPrice && meal.price_idr !== null ? (
                  <>
                    <span aria-hidden="true">·</span>
                    <span className="inline-flex items-center gap-1 font-semibold tabular-nums text-[var(--brand-accent)]">
                      <Wallet
                        className="size-3.5 shrink-0 stroke-[1.75]"
                        aria-hidden="true"
                      />
                      <span>{formatIDR(meal.price_idr)}</span>
                    </span>
                  </>
                ) : null}
              </div>
            </div>

            <div className="flex shrink-0 items-center gap-2">
              <Badge
                className={`h-7 rounded-full border-0 px-3 text-xs font-semibold tabular-nums ${getHealthBadgeClass(meal.health_label)}`}
              >
                {meal.health_score} · {healthLabelText}
              </Badge>

              {onDelete ? (
                <DeleteIconAction
                  mealName={meal.name}
                  isDeleting={isDeleting}
                  deleteLabel={t.meals.deleteButton}
                  deletingLabel={t.meals.deletingButton}
                  cancelLabel={t.common.cancel}
                  buttonClassName="size-11 shrink-0 rounded-full text-[var(--error-text)] hover:bg-[var(--error-bg)] hover:text-[var(--error-text)] shadow-none"
                  onConfirm={handleConfirmDelete}
                />
              ) : null}
            </div>
          </div>

          <div className="grid grid-cols-5 gap-2.5 rounded-2xl bg-muted p-3.5">
            {macros.map((macro) => {
              const Icon = macro.icon;
              return (
                <div key={macro.label} className="space-y-0.5 px-1">
                  <p className="flex items-center gap-1 text-xs text-muted-foreground">
                    <Icon
                      className="size-3.5 shrink-0 stroke-[1.75]"
                      style={{ color: macro.colorVar }}
                      aria-hidden="true"
                    />
                    <span>{macro.label}</span>
                  </p>
                  <p className="whitespace-nowrap text-sm font-semibold tabular-nums text-foreground">
                    {macro.value}{" "}
                    <span className="text-xs font-normal text-muted-foreground">
                      {macro.unit}
                    </span>
                  </p>
                </div>
              );
            })}
          </div>
        </Card>
      ) : null}

      <Dialog open={detailOpen} onOpenChange={setDetailOpen}>
        <div
          className={`flex items-stretch gap-1.5 rounded-2xl border border-border shadow-none transition-colors duration-150 hover:border-[var(--border-strong)] ${
            surface === "canvas" ? "bg-background" : "bg-card"
          } ${showExpandedOnDesktop ? "lg:hidden" : ""}`}
        >
          <DialogTrigger
            render={
              <button
                type="button"
                className="group min-w-0 flex-1 cursor-pointer rounded-2xl px-4 py-3.5 text-left transition-colors duration-150 hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-primary)]"
              />
            }
          >
            <div className="space-y-1.5">
              <div className="flex items-center justify-between gap-2.5">
                <h3 className="min-w-0 truncate text-sm font-semibold leading-snug text-foreground sm:text-[15px]">
                  {meal.name}
                </h3>

                <Badge
                  className={`h-6 shrink-0 rounded-full border-0 px-2.5 text-[11px] font-semibold tabular-nums ${getHealthBadgeClass(meal.health_label)}`}
                >
                  {meal.health_score} · {healthLabelText}
                </Badge>
              </div>

              <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs tabular-nums text-muted-foreground">
                <span className="font-medium text-foreground">
                  {Math.round(meal.calories)} {t.profile.kcalUnit}
                </span>
                <span aria-hidden="true">·</span>
                <span>
                  {t.chat.proteinLabel} {formatMacro(meal.protein_g)}{" "}
                  {t.profile.gramUnit}
                </span>
                {hasPrice && meal.price_idr !== null ? (
                  <>
                    <span aria-hidden="true">·</span>
                    <span className="font-semibold text-[var(--brand-accent)]">
                      {formatIDR(meal.price_idr)}
                    </span>
                  </>
                ) : null}
              </div>

              <div className="flex items-center justify-between gap-3 pt-0.5">
                <p className="min-w-0 truncate text-xs text-muted-foreground">
                  {mealTimeLabel} · {sourceLabel}
                </p>

                <span className="inline-flex shrink-0 items-center gap-0.5 text-xs font-medium text-[var(--brand-primary)] transition-transform duration-150 group-hover:translate-x-0.5">
                  <span>{t.chat.openMealDetailLabel}</span>
                  <ChevronRight
                    className="size-3.5 stroke-[2]"
                    aria-hidden="true"
                  />
                </span>
              </div>
            </div>
          </DialogTrigger>

          {onDelete ? (
            <div className="flex shrink-0 items-center pr-2">
              <DeleteIconAction
                mealName={meal.name}
                isDeleting={isDeleting}
                deleteLabel={t.meals.deleteButton}
                deletingLabel={t.meals.deletingButton}
                cancelLabel={t.common.cancel}
                buttonClassName="size-10 shrink-0 rounded-full text-[var(--error-text)] hover:bg-[var(--error-bg)] hover:text-[var(--error-text)] shadow-none"
                onConfirm={handleConfirmDelete}
              />
            </div>
          ) : null}
        </div>

        <DialogPortal>
          <DialogOverlay className="overflow-hidden overscroll-contain bg-black/40 supports-backdrop-filter:backdrop-blur-md" />
          <DialogPrimitive.Popup
            data-slot="dialog-content"
            className="fixed top-1/2 left-1/2 z-50 grid max-h-[88dvh] w-full max-w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 gap-4 overflow-y-auto overscroll-contain rounded-3xl border border-border bg-card p-5 text-sm text-card-foreground shadow-none ring-0 outline-none duration-150 sm:max-w-lg sm:p-6 data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95"
          >
            <DialogHeader className="gap-2 pr-9">
              <div className="flex flex-wrap items-start justify-between gap-2.5">
                <div className="min-w-0 space-y-1">
                  <DialogTitle className="text-base font-semibold leading-snug text-foreground sm:text-lg">
                    {meal.name}
                  </DialogTitle>
                  <DialogDescription className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <MealTimeIcon
                      className="size-3.5 shrink-0 stroke-[1.75]"
                      aria-hidden="true"
                    />
                    <span>
                      {mealTimeLabel} · {sourceLabel}
                    </span>
                  </DialogDescription>
                </div>

                <Badge
                  className={`h-7 shrink-0 rounded-full border-0 px-3 text-xs font-semibold tabular-nums ${getHealthBadgeClass(meal.health_label)}`}
                >
                  {meal.health_score} · {healthLabelText}
                </Badge>
              </div>
            </DialogHeader>

            <DialogClose
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t.chat.closeMealDetailLabel}
                  className="absolute top-4 right-4 size-8 rounded-full text-muted-foreground shadow-none hover:bg-accent hover:text-foreground"
                />
              }
            >
              <X className="size-4 stroke-[1.75]" aria-hidden="true" />
            </DialogClose>

            {hasPrice && meal.price_idr !== null ? (
              <div className="flex items-center gap-1.5 text-sm">
                <Wallet
                  className="size-3.5 shrink-0 stroke-[1.75] text-[var(--brand-accent)]"
                  aria-hidden="true"
                />
                <span className="text-xs text-muted-foreground">
                  {t.chat.priceLabel}:
                </span>
                <span className="font-semibold tabular-nums text-[var(--brand-accent)]">
                  {formatIDR(meal.price_idr)}
                </span>
              </div>
            ) : null}

            <div className="grid grid-cols-2 gap-2.5 rounded-2xl bg-muted p-3.5 sm:grid-cols-5">
              {macros.map((macro) => {
                const Icon = macro.icon;
                return (
                  <div key={macro.label} className="space-y-0.5 px-1">
                    <p className="flex items-center gap-1 text-xs text-muted-foreground">
                      <Icon
                        className="size-3.5 shrink-0 stroke-[1.75]"
                        style={{ color: macro.colorVar }}
                        aria-hidden="true"
                      />
                      <span>{macro.label}</span>
                    </p>
                    <p className="whitespace-nowrap text-sm font-semibold tabular-nums text-foreground">
                      {macro.value}{" "}
                      <span className="text-xs font-normal text-muted-foreground">
                        {macro.unit}
                      </span>
                    </p>
                  </div>
                );
              })}
            </div>

            {onDelete ? (
              <>
                <Separator className="bg-border" />
                <div className="flex items-center justify-end">
                  <AlertDialog>
                    <AlertDialogTrigger
                      render={
                        <Button
                          type="button"
                          variant="destructive"
                          size="sm"
                          disabled={isDeleting}
                          className="h-9 rounded-full bg-[var(--error-bg)] px-4 text-xs font-medium text-[var(--error-text)] shadow-none hover:bg-[var(--error-bg)]/80"
                        />
                      }
                    >
                      {isDeleting ? (
                        <Loader2
                          className="size-3.5 shrink-0 animate-spin stroke-[1.75]"
                          aria-hidden="true"
                        />
                      ) : (
                        <Trash2
                          className="size-3.5 shrink-0 stroke-[1.75]"
                          aria-hidden="true"
                        />
                      )}
                      <span>
                        {isDeleting
                          ? t.meals.deletingButton
                          : t.meals.deleteButton}
                      </span>
                    </AlertDialogTrigger>
                    <AlertDialogContent className="rounded-3xl border border-border bg-card p-6 shadow-none ring-0">
                      <AlertDialogHeader>
                        <AlertDialogTitle>
                          {t.meals.deleteButton}
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                          {meal.name}
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter className="mx-0 mb-0 rounded-none border-0 bg-transparent p-0 pt-2">
                        <AlertDialogCancel className="h-11 rounded-full px-5 shadow-none">
                          {t.common.cancel}
                        </AlertDialogCancel>
                        <AlertDialogAction
                          variant="destructive"
                          onClick={() => void handleConfirmDelete()}
                          className="h-11 rounded-full bg-[var(--error-bg)] px-5 text-[var(--error-text)] hover:bg-[var(--error-bg)]/80 shadow-none"
                        >
                          {t.meals.deleteButton}
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </>
            ) : null}
          </DialogPrimitive.Popup>
        </DialogPortal>
      </Dialog>
    </>
  );
}
