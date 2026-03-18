"use client";

import { useEffect, useMemo, useState } from "react";
import dynamic from "next/dynamic";

import { AppShell } from "@/components/layout/AppShell";
import { useLanguage } from "@/components/providers/LanguageProvider";
import { ChartCard } from "@/components/charts/ChartCard";
import { Card } from "@/components/ui/Card";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { ScoreBadge } from "@/components/ui/ScoreBadge";
import { Tag } from "@/components/ui/Tag";
import { api } from "@/lib/api/endpoints";
import { pickLocaleText } from "@/lib/i18n";

const ScoreHistoryChart = dynamic(
  () => import("@/components/charts/ScoreHistoryChart").then((module) => module.ScoreHistoryChart),
  { ssr: false },
);

const StrengthWeaknessChart = dynamic(
  () => import("@/components/charts/StrengthWeaknessChart").then((module) => module.StrengthWeaknessChart),
  { ssr: false },
);

type DashboardState = {
  averageScore: number;
  sessionsCompleted: number;
  weakAreas: string[];
  history: { label: string; score: number }[];
  strengthWeakness: { area: string; strength: number; weakness: number }[];
};

function buildStrengthWeaknessData(weakAreas: string[], averageScore: number) {
  if (!weakAreas.length) {
    return [];
  }

  return weakAreas.slice(0, 4).map((area, index) => {
    const weakness = Math.min(95, Math.max(30, 70 - index * 8));
    const strength = Math.min(100, Math.max(5, Math.round(averageScore - index * 4)));

    return {
      area: area.length > 16 ? `${area.slice(0, 16)}…` : area,
      strength,
      weakness,
    };
  });
}

export default function DashboardPage() {
  const { locale } = useLanguage();
  const [state, setState] = useState<DashboardState>({
    averageScore: 0,
    sessionsCompleted: 0,
    weakAreas: [],
    history: [],
    strengthWeakness: [],
  });

  useEffect(() => {
    async function load() {
      try {
        const [progress, history] = await Promise.all([api.getProgress(), api.getSessionHistory()]);
        setState({
          averageScore: Math.round(progress.average_score),
          sessionsCompleted: progress.sessions_completed,
          weakAreas: progress.weak_areas,
          history: history.sessions.map((session, index) => ({ label: `S${index + 1}`, score: session.score })),
          strengthWeakness: buildStrengthWeaknessData(progress.weak_areas, Math.round(progress.average_score)),
        });
      } catch {
        setState((prev) => ({ ...prev, history: [], strengthWeakness: [] }));
      }
    }

    void load();
  }, []);

  const readinessScore = useMemo(
    () => Math.min(100, Math.round(state.averageScore * 0.75 + state.sessionsCompleted * 2.6)),
    [state.averageScore, state.sessionsCompleted],
  );

  const recommendations = state.weakAreas.length
    ? state.weakAreas.map((area) => pickLocaleText(locale, `Latih satu jawaban STAR dengan fokus pada ${area.toLowerCase()}.`, `Practice one STAR answer focused on ${area.toLowerCase()}.`))
    : [pickLocaleText(locale, "Belum ada rekomendasi personal. Selesaikan sesi latihan terlebih dahulu.", "No personalized recommendations yet. Complete a practice session first.")];

  return (
    <AppShell title={pickLocaleText(locale, "Dasbor", "Dashboard")} subtitle={pickLocaleText(locale, "Pantau kesiapan interview dan tingkatkan performa di setiap sesi.", "Track interview readiness and keep improving each session.")}>
      <div className="space-y-5">
        <section className="grid gap-4 md:grid-cols-3">
          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Kesiapan interview", "Interview readiness")}</p>
            <p className="mt-2 text-3xl font-bold text-white">{readinessScore}%</p>
            <div className="mt-3">
              <ProgressBar value={readinessScore} />
            </div>
          </Card>
          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Rata-rata skor", "Average score")}</p>
            <div className="mt-2">
              <ScoreBadge score={state.averageScore} className="text-base" />
            </div>
            <p className="mt-3 text-sm text-white/80">{pickLocaleText(locale, "Berdasarkan sesi terbaru.", "Based on recent sessions.")}</p>
          </Card>
          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Sesi selesai", "Sessions completed")}</p>
            <p className="mt-2 text-3xl font-bold text-white">{state.sessionsCompleted}</p>
            <p className="mt-3 text-sm text-white/80">{pickLocaleText(locale, "Konsistensi meningkatkan kepercayaan diri interview Anda.", "Consistency compounds your interview confidence.")}</p>
          </Card>
        </section>

        <section className="grid gap-4 xl:grid-cols-2">
          <ChartCard title={pickLocaleText(locale, "Riwayat Skor", "Score History")} subtitle={pickLocaleText(locale, "Tren antar sesi", "Session-to-session trend")}>
            {state.history.length > 0 ? (
              <ScoreHistoryChart data={state.history} />
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Belum ada data sesi.", "No session data yet.")}</div>
            )}
          </ChartCard>
          <ChartCard title={pickLocaleText(locale, "Kekuatan vs Kelemahan", "Strength vs Weakness")} subtitle={pickLocaleText(locale, "Dimensi komunikasi inti", "Core communication dimensions")}>
            {state.strengthWeakness.length > 0 ? (
              <StrengthWeaknessChart data={state.strengthWeakness} />
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Belum ada data area lemah.", "No weak-area data yet.")}</div>
            )}
          </ChartCard>
        </section>

        <section className="grid gap-4 xl:grid-cols-2">
          <Card>
            <h3 className="text-base font-semibold text-white">{pickLocaleText(locale, "Sesi latihan terbaru", "Recent practice sessions")}</h3>
            <div className="mt-4 space-y-3">
              {state.history.length > 0 ? (
                state.history.slice(0, 4).map((item, index) => (
                  <div key={`${item.label}-${index}`} className="flex items-center justify-between rounded-[14px] border border-white/10 bg-white/5 px-3 py-2">
                    <p className="text-sm text-white">{pickLocaleText(locale, "Sesi", "Session")} {item.label}</p>
                    <ScoreBadge score={item.score} label="Score" />
                  </div>
                ))
              ) : (
                <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Belum ada riwayat sesi.", "No session history available.")}</p>
              )}
            </div>
          </Card>

          <Card>
            <h3 className="text-base font-semibold text-white">{pickLocaleText(locale, "Rekomendasi tindakan", "Recommended actions")}</h3>
            <div className="mt-4 space-y-3">
              {recommendations.map((recommendation) => (
                <div key={recommendation} className="rounded-[14px] border border-cyan-300/25 bg-cyan-400/10 px-3 py-2 text-sm text-cyan-100">
                  {recommendation}
                </div>
              ))}
            </div>
            <div className="mt-4 flex flex-wrap gap-2">
              {state.weakAreas.length ? state.weakAreas.map((area) => <Tag key={area}>{area}</Tag>) : <Tag>{pickLocaleText(locale, "Tidak ada area lemah terdeteksi", "No weak areas detected")}</Tag>}
            </div>
          </Card>
        </section>
      </div>
    </AppShell>
  );
}
