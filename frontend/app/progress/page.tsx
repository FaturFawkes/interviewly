"use client";

import { useEffect, useMemo, useState } from "react";
import dynamic from "next/dynamic";

import { AppShell } from "@/components/layout/AppShell";
import { useLanguage } from "@/components/providers/LanguageProvider";
import { ChartCard } from "@/components/charts/ChartCard";
import { Card } from "@/components/ui/Card";
import { ProgressBar } from "@/components/ui/ProgressBar";
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

type ProgressState = {
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

export default function ProgressPage() {
  const { locale } = useLanguage();
  const [state, setState] = useState<ProgressState>({
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

  const readiness = useMemo(
    () => Math.min(100, Math.round(state.averageScore * 0.8 + state.sessionsCompleted * 2.2)),
    [state.averageScore, state.sessionsCompleted],
  );

  const recommendedImprovements = state.weakAreas.length
    ? state.weakAreas.map((area) => pickLocaleText(locale, `Lakukan 3 latihan jawaban singkat dengan fokus pada ${area.toLowerCase()}.`, `Run 3 concise practice answers focused on ${area.toLowerCase()}.`))
    : [pickLocaleText(locale, "Belum ada rekomendasi personal. Selesaikan sesi latihan terlebih dahulu.", "No personalized recommendations yet. Complete a practice session first.")];

  return (
    <AppShell title={pickLocaleText(locale, "Analitik Progres", "Progress Analytics")} subtitle={pickLocaleText(locale, "Pantau kesiapan jangka panjang dan fokus pada peningkatan berdampak tinggi.", "Monitor long-term readiness and target high-impact improvements.")}>
      <div className="space-y-4">
        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Skor kesiapan interview", "Interview readiness score")}</p>
            <p className="mt-2 text-4xl font-bold text-white">{readiness}%</p>
            <p className="mt-2 text-sm text-white/80">{pickLocaleText(locale, "Dihitung dari konsistensi dan tren kualitas jawaban.", "Computed from consistency and answer quality trends.")}</p>
            <div className="mt-3">
              <ProgressBar value={readiness} />
            </div>
          </Card>

          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Fokus perbaikan", "Improvement focus")}</p>
            <div className="mt-3 flex flex-wrap gap-2">
              {state.weakAreas.length > 0 ? state.weakAreas.map((area) => <Tag key={area}>{area}</Tag>) : <Tag>{pickLocaleText(locale, "Pertahankan momentum", "Maintain momentum")}</Tag>}
            </div>
          </Card>
        </section>

        <section className="grid gap-4 xl:grid-cols-2">
          <ChartCard title={pickLocaleText(locale, "Riwayat Skor", "Score History")} subtitle={pickLocaleText(locale, "Tren kualitas rata-rata antar sesi", "Average quality trend over sessions")}>
            {state.history.length > 0 ? (
              <ScoreHistoryChart data={state.history} />
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Belum ada data sesi.", "No session data yet.")}</div>
            )}
          </ChartCard>
          <ChartCard title={pickLocaleText(locale, "Kekuatan vs Kelemahan", "Strength vs Weakness")} subtitle={pickLocaleText(locale, "Profil kapabilitas komunikasi", "Communication capability profile")}>
            {state.strengthWeakness.length > 0 ? (
              <StrengthWeaknessChart data={state.strengthWeakness} />
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Belum ada data area lemah.", "No weak-area data yet.")}</div>
            )}
          </ChartCard>
        </section>

        <Card>
          <h3 className="text-base font-semibold text-white">{pickLocaleText(locale, "Area perbaikan yang direkomendasikan", "Recommended improvement areas")}</h3>
          <div className="mt-4 grid gap-3 md:grid-cols-2">
            {recommendedImprovements.map((item) => (
              <div key={item} className="rounded-[14px] border border-cyan-300/30 bg-cyan-400/10 px-3 py-2 text-sm text-cyan-100">
                {item}
              </div>
            ))}
          </div>
        </Card>
      </div>
    </AppShell>
  );
}
