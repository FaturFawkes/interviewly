"use client";

import { useLanguage } from "@/components/providers/LanguageProvider";
import { cn } from "@/lib/utils";

export function LanguageSwitcher() {
  const { locale, setLocale } = useLanguage();

  return (
    <div className="inline-flex items-center gap-1 rounded-full border border-white/15 bg-white/5 p-1">
      <button
        type="button"
        onClick={() => setLocale("id")}
        className={cn(
          "rounded-full px-2.5 py-1 text-xs transition",
          locale === "id" ? "bg-cyan-400/20 text-cyan-100" : "text-white/65 hover:text-white",
        )}
      >
        ID
      </button>
      <button
        type="button"
        onClick={() => setLocale("en")}
        className={cn(
          "rounded-full px-2.5 py-1 text-xs transition",
          locale === "en" ? "bg-cyan-400/20 text-cyan-100" : "text-white/65 hover:text-white",
        )}
      >
        EN
      </button>
    </div>
  );
}
