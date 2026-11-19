import { enDictionary } from "@/lib/i18n/en";
import { idDictionary } from "@/lib/i18n/id";
import type { Dictionary } from "@/lib/i18n/types";
import type { Language } from "@/lib/types";

export const dictionaries: Record<Language, Dictionary> = {
  id: idDictionary,
  en: enDictionary,
};

export type { Dictionary };
