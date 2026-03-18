"use client";

import { createContext, useContext, useEffect, useMemo, useState } from "react";

import { DEFAULT_LOCALE, type Locale } from "@/lib/i18n";

type LanguageContextValue = {
  locale: Locale;
  setLocale: (value: Locale) => void;
};

const STORAGE_KEY = "interviewly_locale";

const LanguageContext = createContext<LanguageContextValue | null>(null);

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(DEFAULT_LOCALE);

  useEffect(() => {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored === "id" || stored === "en") {
      setLocaleState(stored);
    }
  }, []);

  function setLocale(value: Locale) {
    setLocaleState(value);
    window.localStorage.setItem(STORAGE_KEY, value);
  }

  const contextValue = useMemo<LanguageContextValue>(() => ({ locale, setLocale }), [locale]);

  return <LanguageContext.Provider value={contextValue}>{children}</LanguageContext.Provider>;
}

export function useLanguage() {
  const context = useContext(LanguageContext);
  if (!context) {
    throw new Error("useLanguage must be used within LanguageProvider");
  }

  return context;
}
