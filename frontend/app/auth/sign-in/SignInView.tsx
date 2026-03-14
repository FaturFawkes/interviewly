"use client";

import { useEffect, useState } from "react";
import { signIn } from "next-auth/react";

import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";

type SignInProvider = "google" | "azure-ad";

export function SignInView({ callbackUrl }: { callbackUrl: string }) {
  const publicApiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api-proxy";
  const [loadingProvider, setLoadingProvider] = useState<SignInProvider | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [otp, setOTP] = useState("");
  const [otpRequested, setOtpRequested] = useState(false);
  const [resendCooldown, setResendCooldown] = useState(0);
  const [credentialsLoading, setCredentialsLoading] = useState<"login" | "register" | "verify-otp" | "resend-otp" | null>(null);
  const [credentialsError, setCredentialsError] = useState<string | null>(null);
  const [credentialsInfo, setCredentialsInfo] = useState<string | null>(null);

  useEffect(() => {
    if (resendCooldown <= 0) {
      return;
    }

    const intervalId = window.setInterval(() => {
      setResendCooldown((current) => (current > 0 ? current - 1 : 0));
    }, 1000);

    return () => window.clearInterval(intervalId);
  }, [resendCooldown]);

  function formatCooldown(seconds: number): string {
    const minutes = Math.floor(seconds / 60);
    const remainder = seconds % 60;
    return `${minutes}:${remainder.toString().padStart(2, "0")}`;
  }

  async function handleSignIn(provider: SignInProvider) {
    setLoadingProvider(provider);
    await signIn(provider, { callbackUrl });
    setLoadingProvider(null);
  }

  async function handleCredentials(mode: "login") {
    setCredentialsError(null);
    setCredentialsInfo(null);
    setCredentialsLoading(mode);

    const result = await signIn("credentials", {
      redirect: false,
      callbackUrl,
      email,
      password,
    });

    if (!result || result.error) {
      setCredentialsError("Invalid email or password.");
      setCredentialsLoading(null);
      return;
    }

    window.location.href = callbackUrl;
  }

  async function handleRequestOTP() {
    setCredentialsError(null);
    setCredentialsInfo(null);
    setCredentialsLoading("register");

    try {
      const response = await fetch(`${publicApiBaseUrl}/auth/register`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email,
          full_name: fullName,
          password,
        }),
      });

      const payload = (await response.json().catch(() => ({}))) as { error?: string; expires_in?: number; resend_available_in?: number; retry_after?: number };
      if (!response.ok) {
        if (payload.retry_after && payload.retry_after > 0) {
          setResendCooldown(payload.retry_after);
        }
        throw new Error(payload.error ?? "Failed to request OTP.");
      }

      const expiryMinutes = payload.expires_in ? Math.round(payload.expires_in / 60) : 10;
      setOtpRequested(true);
      setResendCooldown(payload.resend_available_in ?? 300);
      setCredentialsInfo(`OTP sent to your email. Expires in ${expiryMinutes} minutes.`);
    } catch (error) {
      setCredentialsError(error instanceof Error ? error.message : "Failed to request OTP.");
    } finally {
      setCredentialsLoading(null);
    }
  }

  async function handleVerifyOTP() {
    setCredentialsError(null);
    setCredentialsInfo(null);
    setCredentialsLoading("verify-otp");

    try {
      const response = await fetch(`${publicApiBaseUrl}/auth/register/verify`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email,
          otp,
        }),
      });

      const payload = (await response.json().catch(() => ({}))) as { error?: string; message?: string };
      if (!response.ok) {
        throw new Error(payload.error ?? "OTP verification failed.");
      }

      setFullName("");
      setEmail("");
      setPassword("");
      setOTP("");
      setOtpRequested(false);
      setResendCooldown(0);
      setCredentialsInfo("Registration successful. Please login with your email and password.");
    } catch (error) {
      setCredentialsError(error instanceof Error ? error.message : "OTP verification failed.");
    } finally {
      setCredentialsLoading(null);
    }
  }

  async function handleResendOTP() {
    setCredentialsError(null);
    setCredentialsInfo(null);
    setCredentialsLoading("resend-otp");

    try {
      const response = await fetch(`${publicApiBaseUrl}/auth/register/resend`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ email }),
      });

      const payload = (await response.json().catch(() => ({}))) as { error?: string; expires_in?: number; resend_available_in?: number; retry_after?: number };
      if (!response.ok) {
        if (payload.retry_after && payload.retry_after > 0) {
          setResendCooldown(payload.retry_after);
        }
        throw new Error(payload.error ?? "Failed to resend OTP.");
      }

      setResendCooldown(payload.resend_available_in ?? 300);
      setCredentialsInfo("OTP resent. Please check your inbox and spam folder.");
    } catch (error) {
      setCredentialsError(error instanceof Error ? error.message : "Failed to resend OTP.");
    } finally {
      setCredentialsLoading(null);
    }
  }

  return (
    <main className="section-shell flex min-h-screen items-center justify-center py-10">
      <Card className="w-full max-w-md space-y-5">
        <div>
          <h1 className="text-2xl font-semibold text-white">Sign in to AI Interview Coach</h1>
          <p className="mt-2 text-sm text-[var(--color-text-muted)]">Continue with your Google or Microsoft account.</p>
        </div>

        <div className="space-y-3">
          <Button
            className="w-full"
            onClick={() => void handleSignIn("google")}
            disabled={loadingProvider !== null}
          >
            {loadingProvider === "google" ? "Redirecting..." : "Continue with Google"}
          </Button>

          <Button
            variant="secondary"
            className="w-full"
            onClick={() => void handleSignIn("azure-ad")}
            disabled={loadingProvider !== null}
          >
            {loadingProvider === "azure-ad" ? "Redirecting..." : "Continue with Microsoft"}
          </Button>

          <div className="my-2 border-t border-white/10" />

          <Input
            value={fullName}
            onChange={(event) => setFullName(event.target.value)}
            placeholder="Full name (for register)"
          />
          <Input
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            placeholder="Email"
          />
          <Input
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="Password"
          />

          <Button
            className="w-full"
            onClick={() => void handleRequestOTP()}
            disabled={credentialsLoading !== null || !email.trim() || password.length < 8}
          >
            {credentialsLoading === "register" ? "Sending OTP..." : "Register with Email"}
          </Button>

          {otpRequested && (
            <>
              <Input
                value={otp}
                onChange={(event) => setOTP(event.target.value)}
                placeholder="Input OTP from email"
              />
              <div className="-mt-1 text-xs text-[var(--color-text-muted)]">
                Didn&apos;t receive the email?{" "}
                <button
                  type="button"
                  onClick={() => void handleResendOTP()}
                  disabled={credentialsLoading !== null || resendCooldown > 0 || !email.trim()}
                  className="font-medium text-cyan-300 underline decoration-cyan-300/70 underline-offset-2 disabled:cursor-not-allowed disabled:text-white/40 disabled:no-underline"
                >
                  {credentialsLoading === "resend-otp"
                    ? "Sending..."
                    : resendCooldown > 0
                      ? `Resend OTP in ${formatCooldown(resendCooldown)}`
                      : "Resend OTP"}
                </button>
              </div>
              <Button
                className="w-full"
                onClick={() => void handleVerifyOTP()}
                disabled={credentialsLoading !== null || !otp.trim()}
              >
                {credentialsLoading === "verify-otp" ? "Verifying OTP..." : "Verify OTP"}
              </Button>
            </>
          )}

          <Button
            variant="secondary"
            className="w-full"
            onClick={() => void handleCredentials("login")}
            disabled={credentialsLoading !== null || !email.trim() || !password}
          >
            {credentialsLoading === "login" ? "Signing in..." : "Login with Email"}
          </Button>

          {credentialsError && <p className="text-sm text-red-300">{credentialsError}</p>}
          {credentialsInfo && <p className="text-sm text-cyan-200">{credentialsInfo}</p>}
        </div>
      </Card>
    </main>
  );
}
