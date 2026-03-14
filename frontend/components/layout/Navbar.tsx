"use client";

import Link from "next/link";
import { signOut, useSession } from "next-auth/react";
import { Sparkles } from "lucide-react";

import { GradientButton } from "@/components/ui/GradientButton";

export function Navbar() {
  const { status } = useSession();
  const authenticated = status === "authenticated";

  return (
    <header className="relative z-10 mx-auto flex max-w-7xl items-center justify-between px-8 py-5">
      <Link href="/" className="flex items-center gap-3">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-purple-500 to-cyan-400">
          <Sparkles className="h-5 w-5 text-white" />
        </div>
        <span className="text-white tracking-tight">AI Interview Coach</span>
      </Link>

      <div className="hidden md:flex items-center gap-8">
        <a href="#features" className="text-white/60 hover:text-white transition-colors text-sm">Features</a>
        <a href="#testimonials" className="text-white/60 hover:text-white transition-colors text-sm">Testimonials</a>
        <a href="#pricing" className="text-white/60 hover:text-white transition-colors text-sm">Pricing</a>
      </div>

      <div className="flex items-center gap-2">
        {authenticated ? (
          <GradientButton size="sm" onClick={() => void signOut({ callbackUrl: "/" })}>Logout</GradientButton>
        ) : (
          <Link href="/dashboard">
            <GradientButton size="sm">Get Started</GradientButton>
          </Link>
        )}
      </div>
    </header>
  );
}
