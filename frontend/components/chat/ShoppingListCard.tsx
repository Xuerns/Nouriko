"use client";

import { useI18n } from "@/components/providers/I18nProvider";
import { formatIDR, formatMacro } from "@/components/chat/format";
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
    <section className="rounded-lg border border-neutral-300 dark:border-neutral-700 p-4 space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
          {t.chat.shoppingListHeading}
        </h3>
        <span className="rounded border border-neutral-300 dark:border-neutral-700 px-2 py-0.5 font-mono text-xs text-neutral-700 dark:text-neutral-300">
          {t.chat.estimatedTag}
        </span>
      </div>

      <div className="overflow-x-auto rounded-md border border-neutral-300 dark:border-neutral-700">
        <table className="w-full min-w-[460px] border-collapse text-left text-sm">
          <thead>
            <tr className="border-b border-neutral-300 dark:border-neutral-700 font-mono text-xs uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
              <th className="px-3 py-2 font-medium">{t.chat.shoppingItemCol}</th>
              <th className="px-3 py-2 font-medium">{t.chat.shoppingUnitCol}</th>
              <th className="px-3 py-2 font-medium text-right">
                {t.chat.shoppingPriceCol}
              </th>
              <th className="px-3 py-2 font-medium">
                {t.chat.shoppingMacroCol}
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-300 dark:divide-neutral-700">
            {items.map((item, idx) => (
              <tr key={`${item.name}-${idx}`}>
                <td className="px-3 py-2 font-medium text-neutral-900 dark:text-neutral-100">
                  {item.name}
                </td>
                <td className="px-3 py-2 text-neutral-600 dark:text-neutral-400">
                  {item.unit}
                </td>
                <td className="px-3 py-2 text-right font-mono text-neutral-900 dark:text-neutral-100">
                  {formatIDR(item.estimated_price_idr)}
                </td>
                <td className="px-3 py-2 font-mono text-xs text-neutral-600 dark:text-neutral-400">
                  {Math.round(item.calories)} {t.profile.kcalUnit} |{" "}
                  {macroShort.protein} {formatMacro(item.protein_g)}
                  {t.profile.gramUnit} | {macroShort.carbs}{" "}
                  {formatMacro(item.carbs_g)}
                  {t.profile.gramUnit} | {macroShort.fat}{" "}
                  {formatMacro(item.fat_g)}
                  {t.profile.gramUnit} | {macroShort.fiber}{" "}
                  {formatMacro(item.fiber_g)}
                  {t.profile.gramUnit}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-between border-t border-neutral-300 dark:border-neutral-700 pt-2.5 text-sm">
        <span className="font-medium text-neutral-700 dark:text-neutral-300">
          {t.chat.shoppingTotalLabel}
        </span>
        <span className="font-mono font-semibold text-neutral-900 dark:text-neutral-100">
          {formatIDR(totalIDR)}
        </span>
      </div>
    </section>
  );
}
