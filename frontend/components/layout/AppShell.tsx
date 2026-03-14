"use client";

import { useState } from "react";

import { DashboardHeader } from "@/components/layout/DashboardHeader";
import { Sidebar } from "@/components/layout/Sidebar";

type AppShellProps = {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
};

export function AppShell({ title, subtitle, children }: AppShellProps) {
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <div className="relative flex h-screen w-full bg-[#0B0F14] overflow-hidden">
      <div className="ambient-orb orb-primary -left-24 top-16 h-72 w-72" />
      <div className="ambient-orb orb-cyan right-[-30px] top-[32%] h-72 w-72" />

      <div className="relative flex w-full">
        <Sidebar mobileOpen={mobileOpen} onClose={() => setMobileOpen(false)} />

        <main className="flex-1 overflow-y-auto">
          <div className="p-5 md:p-8 max-w-[1400px] mx-auto">
          <DashboardHeader title={title} subtitle={subtitle} onOpenMenu={() => setMobileOpen(true)} />
          {children}
          </div>
        </main>
      </div>
    </div>
  );
}
