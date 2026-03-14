"use client";

import { useEffect, useMemo, useState } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { ArrowRight, Calendar, Clock, MessageSquare, Target, Zap } from "lucide-react";

import { AppShell } from "@/components/layout/AppShell";
import { ChartCard } from "@/components/charts/ChartCard";
import { Button } from "@/components/ui/Button";
import { GlassCard, GradientBorderCard } from "@/components/ui/GlassCard";
import { ScoreBadge } from "@/components/ui/ScoreBadge";
import { api } from "@/lib/api/endpoints";
import type { AnalyticsPoint, PracticeSession } from "@/lib/api/types";

const ScoreHistoryChart = dynamic(
  () => import("@/components/charts/ScoreHistoryChart").then((module) => module.ScoreHistoryChart),
  { ssr: false },
);

const StrengthWeaknessChart = dynamic(
  () => import("@/components/charts/StrengthWeaknessChart").then((module) => module.StrengthWeaknessChart),
  { ssr: false },
);

type DashboardState = {
  readiness: number;
  averageScore: number;
  avgScoreTrend: number;
  sessionsCompleted: number;
  practiceHours: number;
  weakAreas: string[];
  recommendations: string[];
  sessions: PracticeSession[];
  history: AnalyticsPoint[];
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
  const [state, setState] = useState<DashboardState>({
    readiness: 0,
    averageScore: 0,
    avgScoreTrend: 0,
    sessionsCompleted: 0,
    practiceHours: 0,
    weakAreas: [],
    recommendations: [],
    sessions: [],
    history: [],
    strengthWeakness: [],
  });

  useEffect(() => {
    async function load() {
      try {
        const overview = await api.getAnalyticsOverview();
        setState({
          readiness: overview.interview_readiness,
          averageScore: Math.round(overview.average_score),
          avgScoreTrend: overview.avg_score_trend,
          sessionsCompleted: overview.total_sessions,
          practiceHours: overview.practice_hours,
          weakAreas: overview.weak_areas,
          recommendations: overview.recommendations,
          sessions: overview.recent_sessions,
          history: overview.score_history,
          strengthWeakness: buildStrengthWeaknessData(overview.weak_areas, Math.round(overview.average_score)),
        });
      } catch {
        setState((prev) => ({ ...prev, recommendations: [], sessions: [], history: [], strengthWeakness: [] }));
      }
    }

    void load();
  }, []);

  const recommendations = state.recommendations.length
    ? state.recommendations
    : ["No personalized recommendations yet. Complete a practice session first."];

  const readinessDelta = useMemo(() => `${state.avgScoreTrend >= 0 ? "+" : ""}${state.avgScoreTrend}`, [state.avgScoreTrend]);

  return (
    <AppShell title="Dashboard" subtitle="Here's your interview preparation overview.">
      <div className="space-y-6">
        <section className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 className="text-2xl text-white tracking-tight">Welcome back</h2>
            <p className="text-white/40 mt-1">Keep momentum with focused interview practice.</p>
          </div>
          <Link href="/practice">
            <Button>
              <Zap className="mr-2 h-4 w-4" />
              Start New Session
            </Button>
          </Link>
        </section>

        <section className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <GlassCard className="p-5" glowColor="purple">
            <div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-purple-500 to-purple-700">
              <Target className="h-5 w-5 text-white" />
            </div>
            <p className="text-white/40 text-sm">Overall Score</p>
            <p className="mt-1 text-2xl text-white">{state.averageScore || 0}<span className="text-sm text-white/50">/100</span></p>
            <p className="text-xs text-purple-300 mt-1">{readinessDelta}</p>
          </GlassCard>
          <GlassCard className="p-5">
            <div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-green-500 to-green-700">
              <MessageSquare className="h-5 w-5 text-white" />
            </div>
            <p className="text-white/40 text-sm">Sessions Done</p>
            <p className="mt-1 text-2xl text-white">{state.sessionsCompleted}</p>
            <p className="text-xs text-white/30 mt-1">Completed sessions</p>
          </GlassCard>
          <GlassCard className="p-5">
            <div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-cyan-500 to-cyan-700">
              <Clock className="h-5 w-5 text-white" />
            </div>
            <p className="text-white/40 text-sm">Practice Hours</p>
            <p className="mt-1 text-2xl text-white">{state.practiceHours.toFixed(1)}<span className="text-sm text-white/50">h</span></p>
            <p className="text-xs text-white/30 mt-1">Estimated from sessions</p>
          </GlassCard>
          <GlassCard className="p-5">
            <div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-blue-500 to-blue-700">
              <Calendar className="h-5 w-5 text-white" />
            </div>
            <p className="text-white/40 text-sm">Readiness</p>
            <p className="mt-1 text-2xl text-white">{state.readiness}%</p>
            <p className="text-xs text-white/30 mt-1">Interview confidence index</p>
          </GlassCard>
        </section>

        <section className="grid gap-4 xl:grid-cols-2">
          <ChartCard title="Progress Overview" subtitle="Your score improvement over time">
            {state.history.length > 0 ? (
              <ScoreHistoryChart data={state.history} />
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-muted)]">No session data yet.</div>
            )}
          </ChartCard>
          <ChartCard title="Skill Balance" subtitle="Core communication dimensions">
            {state.strengthWeakness.length > 0 ? (
              <StrengthWeaknessChart data={state.strengthWeakness} />
            ) : (
              <div className="flex h-full items-center justify-center text-sm text-[var(--color-text-muted)]">No weak-area data yet.</div>
            )}
          </ChartCard>
        </section>

        <section className="grid gap-4 xl:grid-cols-2">
          <GlassCard className="p-6 h-full">
            <h3 className="text-white mb-1">Recommended Practice</h3>
            <p className="text-white/40 text-sm mb-5">AI-curated tasks for you</p>
            <div className="mt-4 space-y-3">
              {recommendations.slice(0, 4).map((recommendation, index) => (
                <Link
                  href="/practice"
                  key={`${recommendation}-${index}`}
                  className="w-full flex items-center gap-3 p-3 rounded-xl bg-white/[0.02] border border-white/[0.04] hover:bg-white/[0.05] hover:border-purple-500/20 transition-all text-left group"
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-white/80 text-sm truncate">{recommendation}</p>
                    <p className="text-white/30 text-xs">Focused practice task</p>
                  </div>
                  <ArrowRight className="w-4 h-4 text-white/20 group-hover:text-purple-400 transition-colors shrink-0" />
                </Link>
              ))}
            </div>
          </GlassCard>

          <GradientBorderCard>
            <div className="p-6">
              <div className="flex items-center justify-between mb-5">
                <div>
                  <h3 className="text-white">Recent Sessions</h3>
                  <p className="text-white/40 text-sm mt-0.5">Your latest interview practice results</p>
                </div>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="text-white/30 text-xs uppercase tracking-wider border-b border-white/[0.06]">
                      <th className="text-left pb-3 pr-4">Role</th>
                      <th className="text-left pb-3 pr-4">Company</th>
                      <th className="text-left pb-3 pr-4">Score</th>
                      <th className="text-left pb-3">Mode</th>
                    </tr>
                  </thead>
                  <tbody className="text-sm">
                    {state.sessions.slice(0, 5).map((session) => (
                      <tr key={session.id} className="border-b border-white/[0.04] last:border-0">
                        <td className="py-4 pr-4 text-white/80">{session.target_role || "General Interview"}</td>
                        <td className="py-4 pr-4 text-white/50">{session.target_company || "-"}</td>
                        <td className="py-4 pr-4"><ScoreBadge score={session.score} label="" /></td>
                        <td className="py-4 text-white/50 capitalize">{session.interview_mode || "text"}</td>
                      </tr>
                    ))}
                    {state.sessions.length === 0 && (
                      <tr>
                        <td colSpan={4} className="py-6 text-sm text-white/40">No sessions yet. Start your first interview practice.</td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </GradientBorderCard>
        </section>
      </div>
    </AppShell>
  );
}
