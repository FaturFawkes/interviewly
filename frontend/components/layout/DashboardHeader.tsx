"use client";

import { Menu } from "lucide-react";

import { LanguageSwitcher } from "@/components/layout/LanguageSwitcher";

type DashboardHeaderProps = {
  title: string;
  subtitle?: string;
  onOpenMenu: () => void;
};

export function DashboardHeader({ title, subtitle, onOpenMenu }: DashboardHeaderProps) {
  return (
    <header className="mb-6 flex items-center justify-between rounded-[20px] border border-white/[0.06] bg-white/[0.02] px-4 py-3 backdrop-blur-md sm:px-5">
      <div>
        <h1 className="text-xl font-medium tracking-tight text-white sm:text-2xl">{title}</h1>
        {subtitle && <p className="mt-1 text-sm text-white/40">{subtitle}</p>}
      </div>

      <div className="flex items-center gap-2">
        <LanguageSwitcher />

        <button
          type="button"
          className="rounded-[12px] border border-white/15 p-2 text-white lg:hidden"
          onClick={onOpenMenu}
          aria-label="Open sidebar"
        >
          <Menu className="h-4 w-4" />
        </button>
      </div>
    </header>
  );
}
