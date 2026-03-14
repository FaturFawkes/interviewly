"use client";

import { Menu } from "lucide-react";

type DashboardHeaderProps = {
  title: string;
  subtitle?: string;
  onOpenMenu: () => void;
};

export function DashboardHeader({ title, subtitle, onOpenMenu }: DashboardHeaderProps) {
  return (
    <header className="mb-7 flex items-center justify-between gap-4">
      <div>
        <h1 className="text-2xl text-white tracking-tight">{title}</h1>
        {subtitle && <p className="mt-1 text-white/40">{subtitle}</p>}
      </div>

      <button
        type="button"
        className="rounded-[12px] border border-white/15 p-2 text-white lg:hidden"
        onClick={onOpenMenu}
        aria-label="Open sidebar"
      >
        <Menu className="h-4 w-4" />
      </button>
    </header>
  );
}
