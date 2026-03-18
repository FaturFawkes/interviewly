"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { ArrowRight } from "lucide-react";

import { AppShell } from "@/components/layout/AppShell";
import { useLanguage } from "@/components/providers/LanguageProvider";
import { Card } from "@/components/ui/Card";
import { api } from "@/lib/api/endpoints";
import { pickLocaleText } from "@/lib/i18n";

type DashboardState = {
  sessionsCompleted: number;
  currentPlanID: string;
  trialActive: boolean;
  trialEndsAt?: string;
  totalVoiceMinutes: number;
  usedVoiceMinutes: number;
  remainingVoiceMinutes: number;
  totalSessionsLimit: number;
  usedSessionsInPeriod: number;
  remainingSessionsInPeriod: number;
};

export default function DashboardPage() {
  const { locale } = useLanguage();
  const [state, setState] = useState<DashboardState>({
    sessionsCompleted: 0,
    currentPlanID: "free",
    trialActive: false,
    trialEndsAt: undefined,
    totalVoiceMinutes: 0,
    usedVoiceMinutes: 0,
    remainingVoiceMinutes: 0,
    totalSessionsLimit: 0,
    usedSessionsInPeriod: 0,
    remainingSessionsInPeriod: 0,
  });

  useEffect(() => {
    async function load() {
      try {
        const [progress, subscriptionStatus] = await Promise.all([api.getProgress(), api.getSubscriptionStatus()]);
        setState({
          sessionsCompleted: progress.sessions_completed,
          currentPlanID: subscriptionStatus.plan_id,
          trialActive: subscriptionStatus.trial_active,
          trialEndsAt: subscriptionStatus.trial_ends_at,
          totalVoiceMinutes: subscriptionStatus.total_voice_minutes,
          usedVoiceMinutes: subscriptionStatus.used_voice_minutes,
          remainingVoiceMinutes: subscriptionStatus.remaining_voice_minutes,
          totalSessionsLimit: subscriptionStatus.total_sessions,
          usedSessionsInPeriod: subscriptionStatus.used_sessions,
          remainingSessionsInPeriod: subscriptionStatus.remaining_sessions,
        });
      } catch {
        setState((prev) => ({
          ...prev,
          totalVoiceMinutes: 0,
          usedVoiceMinutes: 0,
          remainingVoiceMinutes: 0,
          totalSessionsLimit: 0,
          usedSessionsInPeriod: 0,
          remainingSessionsInPeriod: 0,
        }));
      }
    }

    void load();
  }, []);

  const planLabel = useMemo(() => {
    switch ((state.currentPlanID || "").toLowerCase()) {
      case "elite":
        return "Elite";
      case "pro":
        return "Pro Career Boost";
      case "starter":
        return "Starter";
      default:
        return pickLocaleText(locale, "Free", "Free");
    }
  }, [locale, state.currentPlanID]);

  const planBadgeClassName = useMemo(() => {
    switch ((state.currentPlanID || "").toLowerCase()) {
      case "elite":
        return "border-violet-300/35 bg-violet-400/15 text-violet-100";
      case "pro":
        return "border-cyan-300/35 bg-cyan-400/15 text-cyan-100";
      case "starter":
        return "border-amber-300/35 bg-amber-400/15 text-amber-100";
      default:
        return "border-emerald-300/35 bg-emerald-400/15 text-emerald-100";
    }
  }, [state.currentPlanID]);

  const trialInfoLabel = useMemo(() => {
    if (!state.trialActive || !state.trialEndsAt) {
      return null;
    }

    try {
      const endsAtDate = new Date(state.trialEndsAt);
      const formatted = endsAtDate.toLocaleString(locale === "id" ? "id-ID" : "en-US", {
        day: "2-digit",
        month: "short",
        hour: "2-digit",
        minute: "2-digit",
      });

      return pickLocaleText(locale, `Trial aktif sampai ${formatted}`, `Trial active until ${formatted}`);
    } catch {
      return pickLocaleText(locale, "Trial aktif", "Trial active");
    }
  }, [locale, state.trialActive, state.trialEndsAt]);

  const sessionLimitLabel = useMemo(() => {
    if (state.totalSessionsLimit < 0) {
      return pickLocaleText(locale, "Tanpa batas", "Unlimited");
    }

    return `${state.usedSessionsInPeriod}/${state.totalSessionsLimit}`;
  }, [locale, state.totalSessionsLimit, state.usedSessionsInPeriod]);

  return (
    <AppShell title={pickLocaleText(locale, "Dasbor", "Dashboard")} subtitle={pickLocaleText(locale, "Ringkasan akun, plan aktif, dan kuota penggunaan Anda.", "Overview of your account, active plan, and usage quotas.")}>
      <div className="space-y-5">
        <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Sesi selesai (total)", "Completed sessions (total)")}</p>
            <p className="mt-2 text-3xl font-bold text-white">{state.sessionsCompleted}</p>
            <p className="mt-3 text-sm text-white/80">{pickLocaleText(locale, "Total histori sesi interview yang sudah Anda selesaikan.", "Total historical interview sessions you have completed.")}</p>
          </Card>
          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Kuota voice periode ini", "Voice quota this period")}</p>
            <p className="mt-2 text-3xl font-bold text-white">{state.remainingVoiceMinutes}/{state.totalVoiceMinutes}</p>
            <p className="mt-3 text-sm text-white/80">
              {pickLocaleText(locale, `Terpakai ${state.usedVoiceMinutes} menit voice.`, `${state.usedVoiceMinutes} voice minutes used.`)}
            </p>
          </Card>
          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Kuota sesi periode ini", "Session quota this period")}</p>
            <p className="mt-2 text-3xl font-bold text-white">{sessionLimitLabel}</p>
            <p className="mt-3 text-sm text-white/80">
              {state.totalSessionsLimit < 0
                ? pickLocaleText(locale, "Plan Anda memiliki sesi tanpa batas.", "Your plan includes unlimited sessions.")
                : pickLocaleText(locale, `Sisa ${Math.max(0, state.remainingSessionsInPeriod)} sesi pada periode ini.`, `${Math.max(0, state.remainingSessionsInPeriod)} sessions left in this period.`)}
            </p>
          </Card>
          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Plan aktif", "Current plan")}</p>
            <div className="mt-2">
              <span className={`inline-flex rounded-full border px-3 py-1 text-sm font-semibold ${planBadgeClassName}`}>
                {planLabel}
              </span>
            </div>
            <p className="mt-3 text-sm text-white/80">
              {trialInfoLabel ?? pickLocaleText(locale, "Upgrade kapan saja untuk kuota lebih besar.", "Upgrade anytime for higher limits.")}
            </p>
          </Card>
        </section>

        <section className="grid gap-4 xl:grid-cols-2">
          <Card>
            <h3 className="text-base font-semibold text-white">{pickLocaleText(locale, "Aksi cepat", "Quick actions")}</h3>
            <div className="mt-4 space-y-3">
              <Link href="/practice" className="flex items-center justify-between rounded-[14px] border border-white/10 bg-white/5 px-4 py-3 text-sm text-white/90 hover:bg-white/10">
                <span>{pickLocaleText(locale, "Mulai latihan interview", "Start interview practice")}</span>
                <ArrowRight className="h-4 w-4" />
              </Link>
              <Link href="/upload" className="flex items-center justify-between rounded-[14px] border border-white/10 bg-white/5 px-4 py-3 text-sm text-white/90 hover:bg-white/10">
                <span>{pickLocaleText(locale, "Unggah atau perbarui resume", "Upload or update resume")}</span>
                <ArrowRight className="h-4 w-4" />
              </Link>
              <Link href="/analytics" className="flex items-center justify-between rounded-[14px] border border-cyan-300/20 bg-cyan-400/10 px-4 py-3 text-sm text-cyan-100 hover:bg-cyan-400/15">
                <span>{pickLocaleText(locale, "Lihat analisis hasil interview", "View interview result analysis")}</span>
                <ArrowRight className="h-4 w-4" />
              </Link>
            </div>
          </Card>

          <Card>
            <h3 className="text-base font-semibold text-white">{pickLocaleText(locale, "Tentang halaman Analytics", "About the Analytics page")}</h3>
            <div className="mt-4 space-y-3">
              <div className="rounded-[14px] border border-white/10 bg-white/5 px-4 py-3 text-sm text-white/85">
                {pickLocaleText(locale, "Analytics berfokus pada analisis hasil interview: tren skor, strength vs weakness, dan rekomendasi latihan berbasis performa.", "Analytics focuses on interview result analysis: score trends, strength vs weakness, and performance-based practice recommendations.")}
              </div>
              <div className="rounded-[14px] border border-white/10 bg-white/5 px-4 py-3 text-sm text-white/85">
                {pickLocaleText(locale, "Dashboard ini hanya menampilkan ringkasan umum akun, plan, dan kuota penggunaan agar tidak tumpang tindih dengan Analytics.", "This dashboard only shows general account, plan, and usage summary to avoid overlap with Analytics.")}
              </div>
            </div>
          </Card>
        </section>
      </div>
    </AppShell>
  );
}
