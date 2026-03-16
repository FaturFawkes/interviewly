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
    <div className="relative min-h-screen overflow-hidden bg-[#0B0F14]">
      <div className="ambient-orb orb-primary -left-24 top-20 h-64 w-64" />
      <div className="ambient-orb orb-cyan -right-16 top-1/3 h-72 w-72" />

      <div className="relative mx-auto flex min-h-screen w-full max-w-[1440px] gap-4 p-4 lg:gap-5 lg:p-5">
        <Sidebar mobileOpen={mobileOpen} onClose={() => setMobileOpen(false)} />

        <main className="flex-1 overflow-y-auto lg:pl-1">
          <DashboardHeader title={title} subtitle={subtitle} onOpenMenu={() => setMobileOpen(true)} />
          <div className="pb-8">{children}</div>
        </main>
      </div>
    </div>
  );
}
