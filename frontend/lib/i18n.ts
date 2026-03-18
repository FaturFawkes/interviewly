export type Locale = "id" | "en";

export const DEFAULT_LOCALE: Locale = "id";

export function pickLocaleText(locale: Locale, indonesian: string, english: string): string {
  return locale === "id" ? indonesian : english;
}
