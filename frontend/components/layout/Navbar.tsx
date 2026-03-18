"use client";

import Link from "next/link";
import { signOut, useSession } from "next-auth/react";
import { Sparkles } from "lucide-react";

import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher";
import { useLanguage } from "@/components/providers/LanguageProvider";
import { Button } from "@/components/ui/Button";
import { pickLocaleText } from "@/lib/i18n";

export function Navbar() {
  const { status } = useSession();
  const { locale } = useLanguage();
  const authenticated = status === "authenticated";

  const dashboardText = pickLocaleText(locale, "Dasbor", "Dashboard");
  const logoutText = pickLocaleText(locale, "Keluar", "Logout");
  const loginText = pickLocaleText(locale, "Masuk", "Login");

  return (
    <header className="section-shell relative z-10 flex items-center justify-between py-6">
      <Link href="/" className="inline-flex items-center gap-2 text-sm font-semibold tracking-wide text-white">
        <Sparkles className="h-4 w-4 text-cyan-300" />
        <span className="gradient-text text-base">AI Interview Coach</span>
      </Link>

      <div className="flex items-center gap-2">
        <LanguageSwitcher />
        <Link href="/dashboard">
          <Button variant="ghost" className="hidden sm:inline-flex">
            {dashboardText}
          </Button>
        </Link>

        {authenticated ? (
          <Button variant="secondary" onClick={() => void signOut({ callbackUrl: "/" })}>
            {logoutText}
          </Button>
        ) : (
          <Link href="/auth/sign-in">
            <Button variant="secondary">{loginText}</Button>
          </Link>
        )}
      </div>
    </header>
  );
}
