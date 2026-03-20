"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { signOut } from "next-auth/react";
import {
  BarChart3,
  Brain,
  ChevronLeft,
  CreditCard,
  LayoutDashboard,
  LogOut,
  MessageSquare,
  Settings,
  Sparkles,
  Upload,
  X,
} from "lucide-react";

import { useLanguage } from "@/components/providers/LanguageProvider";
import { setStoredAuthToken } from "@/lib/auth/token-provider";
import { pickLocaleText } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const navItems = [
  { href: "/dashboard", labelID: "Dasbor", labelEN: "Dashboard", icon: LayoutDashboard },
  { href: "/upload", labelID: "Unggah Resume (CV)", labelEN: "Upload Resume (CV)", icon: Upload },
  { href: "/practice", labelID: "Latihan", labelEN: "Practice", icon: MessageSquare },
  { href: "/review", labelID: "Review Coach", labelEN: "Review Coach", icon: Brain },
  { href: "/analytics", labelID: "Analisis Interview", labelEN: "Interview Analytics", icon: BarChart3 },
  { href: "/billing", labelID: "Billing & Top-Up", labelEN: "Billing & Top-Up", icon: CreditCard },
];

type SidebarProps = {
  mobileOpen: boolean;
  onClose: () => void;
};

export function Sidebar({ mobileOpen, onClose }: SidebarProps) {
  const pathname = usePathname();
  const { locale } = useLanguage();
  const [collapsed, setCollapsed] = useState(false);

  async function handleSignOut(): Promise<void> {
    if (typeof window !== "undefined") {
      const keysToRemove: string[] = [];
      for (let index = 0; index < window.sessionStorage.length; index += 1) {
        const key = window.sessionStorage.key(index);
        if (key?.startsWith("interview-")) {
          keysToRemove.push(key);
        }
      }

      keysToRemove.forEach((key) => {
        window.sessionStorage.removeItem(key);
      });
    }

    setStoredAuthToken(null);
    onClose();
    await signOut({ callbackUrl: "/" });
  }

  return (
    <>
      <aside
        className={cn(
          "fixed inset-y-4 left-4 z-40 flex flex-col overflow-hidden rounded-[20px] border border-white/[0.06] bg-[#0A0E13]/80 backdrop-blur-xl transition-all duration-300 lg:static lg:inset-auto lg:translate-x-0",
          collapsed ? "w-[72px]" : "w-[260px]",
          mobileOpen ? "translate-x-0" : "-translate-x-[115%]",
        )}
      >
        <div className="flex items-center gap-3 border-b border-white/[0.06] p-5">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-linear-to-br from-purple-500 to-cyan-400 shrink-0">
            <Sparkles className="h-5 w-5 text-white" />
          </div>
          {!collapsed && <h2 className="text-base text-white tracking-tight">AI Interview Coach</h2>}

          <button
            type="button"
            onClick={() => setCollapsed((value) => !value)}
            className="ml-auto rounded-md p-1 text-white/40 transition-colors hover:text-white/80"
            aria-label={pickLocaleText(locale, "Ciutkan sidebar", "Collapse sidebar")}
          >
            <ChevronLeft className={cn("h-4 w-4 transition-transform", collapsed && "rotate-180")} />
          </button>

          <button
            type="button"
            onClick={onClose}
            className="rounded-md p-1 text-white/70 hover:bg-white/10 hover:text-white lg:hidden"
            aria-label={pickLocaleText(locale, "Tutup sidebar", "Close sidebar")}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <nav className="flex-1 space-y-1 p-3">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = pathname === item.href;
            const label = pickLocaleText(locale, item.labelID, item.labelEN);

            return (
              <Link
                key={item.href}
                href={item.href}
                onClick={onClose}
                className={cn(
                  "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm transition-all duration-200",
                  active
                    ? "bg-linear-to-r from-purple-500/15 to-blue-500/10 text-white shadow-[0_0_15px_rgba(139,92,246,0.1)]"
                    : "text-white/50 hover:bg-white/[0.04] hover:text-white/80",
                )}
              >
                <Icon className={cn("h-5 w-5 shrink-0", active && "text-purple-400")} />
                {!collapsed && <span className="whitespace-nowrap">{label}</span>}
                {active && !collapsed && (
                  <div className="ml-auto h-1.5 w-1.5 rounded-full bg-purple-400 shadow-[0_0_6px_rgba(139,92,246,0.8)]" />
                )}
              </Link>
            );
          })}
        </nav>

        <div className="space-y-1 border-t border-white/[0.06] p-3">
          <button
            type="button"
            className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-white/50 transition-all hover:bg-white/[0.04] hover:text-white/80"
          >
            <Settings className="h-5 w-5 shrink-0" />
            {!collapsed && <span>{pickLocaleText(locale, "Pengaturan", "Settings")}</span>}
          </button>

          <button
            type="button"
            onClick={() => {
              void handleSignOut();
            }}
            className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-white/50 transition-all hover:bg-red-500/5 hover:text-red-400"
          >
            <LogOut className="h-5 w-5 shrink-0" />
            {!collapsed && <span>{pickLocaleText(locale, "Keluar", "Sign Out")}</span>}
          </button>
        </div>
      </aside>

      {mobileOpen && (
        <button
          type="button"
          className="fixed inset-0 z-30 bg-black/50 lg:hidden"
          onClick={onClose}
          aria-label={pickLocaleText(locale, "Tutup latar sidebar", "Close sidebar backdrop")}
        />
      )}
    </>
  );
}
