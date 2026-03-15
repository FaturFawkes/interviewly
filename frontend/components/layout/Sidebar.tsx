"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { signOut } from "next-auth/react";
import { BarChart3, ChevronLeft, LayoutDashboard, LogOut, Mic, Settings, Sparkles, Upload } from "lucide-react";
import { useState } from "react";

import { cn } from "@/lib/utils";

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard, match: ["/dashboard"] },
  { href: "/upload", label: "Upload CV", icon: Upload, match: ["/upload"] },
  { href: "/practice", label: "Practice", icon: Mic, match: ["/practice"] },
  { href: "/analytics", label: "Analytics", icon: BarChart3, match: ["/analytics"] },
];

type SidebarProps = {
  mobileOpen: boolean;
  onClose: () => void;
};

export function Sidebar({ mobileOpen, onClose }: SidebarProps) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  async function handleLogout() {
    if (loggingOut) {
      return;
    }

    setLoggingOut(true);

    try {
      onClose();
      await signOut({ callbackUrl: "/" });
    } finally {
      setLoggingOut(false);
    }
  }

  return (
    <>
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-40 flex flex-col bg-[#0A0E13]/80 backdrop-blur-xl border-r border-white/[0.06] transition-all duration-300 lg:static lg:translate-x-0",
          collapsed ? "w-[72px]" : "w-[260px]",
          mobileOpen ? "translate-x-0" : "-translate-x-[115%]",
        )}
      >
        <div className="p-5 flex items-center gap-3 border-b border-white/[0.06]">
          <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-purple-500 to-cyan-400 flex items-center justify-center shrink-0">
            <Sparkles className="w-5 h-5 text-white" />
          </div>
          {!collapsed && <span className="text-white tracking-tight whitespace-nowrap">AI Interview Coach</span>}
          <button
            type="button"
            onClick={() => {
              if (window.innerWidth >= 1024) {
                setCollapsed((prev) => !prev);
                return;
              }
              onClose();
            }}
            className="ml-auto text-white/40 hover:text-white/80 transition-colors"
            aria-label="Toggle sidebar"
          >
            <ChevronLeft className={cn("w-4 h-4 transition-transform", collapsed && "rotate-180")} />
          </button>
        </div>

        <nav className="flex-1 p-3 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = item.match.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`));

            return (
              <Link
                key={item.href}
                href={item.href}
                onClick={onClose}
                className={cn(
                  "w-full flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-200",
                  active
                    ? "bg-gradient-to-r from-purple-500/15 to-blue-500/10 text-white shadow-[0_0_15px_rgba(139,92,246,0.1)]"
                    : "text-white/50 hover:text-white/80 hover:bg-white/[0.04]",
                )}
              >
                <Icon className="w-4 h-4 shrink-0" />
                {!collapsed && <span className="text-sm whitespace-nowrap">{item.label}</span>}
              </Link>
            );
          })}
        </nav>

        <div className="p-3 border-t border-white/[0.06] space-y-1">
          <button
            type="button"
            onClick={onClose}
            className="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-200 text-white/50 hover:text-white/80 hover:bg-white/[0.04]"
            aria-label="Open settings"
          >
            <Settings className="w-4 h-4 shrink-0" />
            {!collapsed && <span className="text-sm whitespace-nowrap">Setting</span>}
          </button>

          <button
            type="button"
            onClick={() => void handleLogout()}
            disabled={loggingOut}
            className="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-200 text-white/50 hover:text-white/80 hover:bg-white/[0.04] disabled:cursor-not-allowed disabled:opacity-60"
            aria-label="Logout"
          >
            <LogOut className="w-4 h-4 shrink-0" />
            {!collapsed && <span className="text-sm whitespace-nowrap">{loggingOut ? "Logging out..." : "Logout"}</span>}
          </button>
        </div>
      </aside>

      {mobileOpen && (
        <button
          type="button"
          className="fixed inset-0 z-30 bg-black/50 lg:hidden"
          onClick={onClose}
          aria-label="Close sidebar backdrop"
        />
      )}
    </>
  );
}
