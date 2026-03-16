"use client";

import Link from "next/link";
import { signOut, useSession } from "next-auth/react";
import { Sparkles } from "lucide-react";

import { Button } from "@/components/ui/Button";

export function Navbar() {
  const { status } = useSession();
  const authenticated = status === "authenticated";

  return (
    <header className="section-shell relative z-10 flex items-center justify-between py-6">
      <Link href="/" className="inline-flex items-center gap-2 text-sm font-semibold tracking-wide text-white">
        <Sparkles className="h-4 w-4 text-cyan-300" />
        <span className="gradient-text text-base">AI Interview Coach</span>
      </Link>

      <div className="flex items-center gap-2">
        <Link href="/dashboard">
          <Button variant="ghost" className="hidden sm:inline-flex">
            Dashboard
          </Button>
        </Link>

        {authenticated ? (
          <Button variant="secondary" onClick={() => void signOut({ callbackUrl: "/" })}>
            Logout
          </Button>
        ) : (
          <Link href="/auth/sign-in">
            <Button variant="secondary">Login</Button>
          </Link>
        )}
      </div>
    </header>
  );
}
