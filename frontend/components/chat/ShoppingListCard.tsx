"use client";

import { useI18n } from "@/components/providers/I18nProvider";
import { formatIDR, formatMacro } from "@/components/chat/format";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { ShoppingListItem } from "@/lib/types";

interface ShoppingListCardProps {
  items: ShoppingListItem[];
}

export function ShoppingListCard({ items }: ShoppingListCardProps) {
  const { t, language } = useI18n();
  const macroShort =
    language === "en"
      ? { protein: "P", carbs: "C", fat: "F", fiber: "Fib" }
      : { protein: "P", carbs: "K", fat: "L", fiber: "S" };

  if (!items || items.length === 0) {
    return null;
  }

  const totalIDR = items.reduce(
    (sum, item) => sum + Math.max(0, item.estimated_price_idr || 0),
    0
  );

  return (
    <Card className="rounded-3xl border border-border bg-card p-5 shadow-none ring-0 gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-base font-semibold text-foreground">
          {t.chat.shoppingListHeading}
        </h3>
        <Badge
          variant="secondary"
          className="rounded-full px-3 text-xs font-medium text-muted-foreground"
        >
          {t.chat.estimatedTag}
        </Badge>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-border">
        <Table className="min-w-[460px]">
          <TableHeader>
            <TableRow className="bg-muted/60 hover:bg-muted/60">
              <TableHead className="px-3 text-[11px] font-semibold uppercase tracking-[0.06em]">
                {t.chat.shoppingItemCol}
              </TableHead>
              <TableHead className="px-3 text-[11px] font-semibold uppercase tracking-[0.06em]">
                {t.chat.shoppingUnitCol}
              </TableHead>
              <TableHead className="px-3 text-right text-[11px] font-semibold uppercase tracking-[0.06em]">
                {t.chat.shoppingPriceCol}
              </TableHead>
              <TableHead className="px-3 text-[11px] font-semibold uppercase tracking-[0.06em]">
                {t.chat.shoppingMacroCol}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((item, idx) => (
              <TableRow key={`${item.name}-${idx}`}>
                <TableCell className="px-3 font-medium text-foreground">
                  {item.name}
                </TableCell>
                <TableCell className="px-3 text-muted-foreground">
                  {item.unit}
                </TableCell>
                <TableCell className="px-3 text-right font-medium tabular-nums text-foreground">
                  {formatIDR(item.estimated_price_idr)}
                </TableCell>
                <TableCell className="px-3 text-xs tabular-nums text-muted-foreground">
                  {Math.round(item.calories)} {t.profile.kcalUnit} ·{" "}
                  {macroShort.protein} {formatMacro(item.protein_g)}
                  {t.profile.gramUnit} · {macroShort.carbs}{" "}
                  {formatMacro(item.carbs_g)}
                  {t.profile.gramUnit} · {macroShort.fat}{" "}
                  {formatMacro(item.fat_g)}
                  {t.profile.gramUnit} · {macroShort.fiber}{" "}
                  {formatMacro(item.fiber_g)}
                  {t.profile.gramUnit}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Separator className="bg-border" />

      <div className="flex items-center justify-between text-sm">
        <span className="font-medium text-muted-foreground">
          {t.chat.shoppingTotalLabel}
        </span>
        <span className="font-semibold tabular-nums text-[var(--brand-accent)]">
          {formatIDR(totalIDR)}
        </span>
      </div>
    </Card>
  );
}
