"use client";

import { useEffect, useMemo, useState } from "react";
import dynamic from "next/dynamic";
import { Brain, Clock, Target, TrendingUp } from "lucide-react";

import { AppShell } from "@/components/layout/AppShell";
import { ChartCard } from "@/components/charts/ChartCard";
import { GlassCard, GradientBorderCard } from "@/components/ui/GlassCard";
import { api } from "@/lib/api/endpoints";
import type { AnalyticsOverview } from "@/lib/api/types";

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
	recentSessions: AnalyticsOverview["recent_sessions"];
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

export default function AnalyticsPage() {
	const [state, setState] = useState<ProgressState>({
		averageScore: 0,
		sessionsCompleted: 0,
		weakAreas: [],
		recentSessions: [],
		history: [],
		strengthWeakness: [],
	});

	useEffect(() => {
		async function load() {
			try {
				const [overview, history] = await Promise.all([
					api.getAnalyticsOverview(),
					api.getSessionHistory(),
				]);

				const historyData = history.sessions
					.slice(-8)
					.map((session, index) => ({ label: `S${index + 1}`, score: session.score }));

				setState({
					averageScore: Math.round(overview.average_score),
					sessionsCompleted: overview.sessions_completed,
					weakAreas: overview.weak_areas,
					recentSessions: overview.recent_sessions,
					history: historyData,
					strengthWeakness: buildStrengthWeaknessData(overview.weak_areas, Math.round(overview.average_score)),
				});
			} catch {
				setState((prev) => ({ ...prev, recentSessions: [], history: [], strengthWeakness: [] }));
			}
		}

		void load();
	}, []);

	const readiness = useMemo(
		() => Math.min(100, Math.round(state.averageScore * 0.8 + state.sessionsCompleted * 2.2)),
		[state.averageScore, state.sessionsCompleted],
	);

	const avgScoreTrend = useMemo(() => {
		if (state.history.length < 2) {
			return 0;
		}

		const first = state.history[0]?.score ?? 0;
		const last = state.history[state.history.length - 1]?.score ?? 0;
		return last - first;
	}, [state.history]);

	const recommendedImprovements = state.weakAreas.length
		? state.weakAreas.map((item) => `Focus 2 short STAR answers around ${item.toLowerCase()}.`)
		: ["No personalized recommendations yet. Complete a practice session first."];

	return (
		<AppShell title="Progress Analytics" subtitle="Track your improvement and identify areas for growth">
			<div className="space-y-6">
				<section className="grid grid-cols-2 lg:grid-cols-4 gap-4">
					<GlassCard className="p-5" glowColor="purple">
						<div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-linear-to-br from-purple-500 to-purple-700">
							<Target className="h-5 w-5 text-white" />
						</div>
						<p className="text-white/40 text-sm">Interview Readiness</p>
						<p className="text-2xl text-white mt-0.5">{readiness}%</p>
						<p className="text-white/25 text-xs mt-1">Ready for interviews</p>
					</GlassCard>
					<GlassCard className="p-5">
						<div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-linear-to-br from-green-500 to-green-700">
							<TrendingUp className="h-5 w-5 text-white" />
						</div>
						<p className="text-white/40 text-sm">Avg. Score Trend</p>
						<p className="text-2xl text-white mt-0.5">{avgScoreTrend >= 0 ? "+" : ""}{avgScoreTrend}</p>
						<p className="text-white/25 text-xs mt-1">Session trend</p>
					</GlassCard>
					<GlassCard className="p-5">
						<div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-linear-to-br from-blue-500 to-blue-700">
							<Brain className="h-5 w-5 text-white" />
						</div>
						<p className="text-white/40 text-sm">Weak Areas</p>
						<p className="text-2xl text-white mt-0.5">{state.weakAreas.length}</p>
						<p className="text-white/25 text-xs mt-1">Needs practice</p>
					</GlassCard>
					<GlassCard className="p-5">
						<div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-linear-to-br from-cyan-500 to-cyan-700">
							<Clock className="h-5 w-5 text-white" />
						</div>
						<p className="text-white/40 text-sm">Sessions</p>
						<p className="text-2xl text-white mt-0.5">{state.sessionsCompleted}</p>
						<p className="text-white/25 text-xs mt-1">Total completed interviews</p>
					</GlassCard>
				</section>

				<section className="grid gap-4 xl:grid-cols-2">
					<ChartCard title="Score History" subtitle="Behavioral and technical trajectory">
						{state.history.length > 0 ? (
							<ScoreHistoryChart data={state.history} />
						) : (
							<div className="flex h-full items-center justify-center text-sm text-muted">No session data yet.</div>
						)}
					</ChartCard>
					<ChartCard title="Strength vs Weakness" subtitle="Capability radar summary">
						{state.strengthWeakness.length > 0 ? (
							<StrengthWeaknessChart data={state.strengthWeakness} />
						) : (
							<div className="flex h-full items-center justify-center text-sm text-muted">No weak-area data yet.</div>
						)}
					</ChartCard>
				</section>

				<GradientBorderCard>
					<div className="p-6">
						<h3 className="text-white mb-1">Practice Recommendations</h3>
						<p className="text-white/40 text-sm mb-5">AI-suggested topics based on weak areas</p>
						<div className="grid md:grid-cols-2 gap-4">
							{recommendedImprovements.map((item, index) => (
								<GlassCard key={`${item}-${index}`} className="p-4 hover:bg-white/4" glowColor="none">
									<div className="flex items-start gap-3">
										<div className="w-10 h-10 rounded-xl bg-purple-500/10 flex items-center justify-center shrink-0">
											<Brain className="w-5 h-5 text-purple-400" />
										</div>
										<div>
											<p className="text-white/80 text-sm">{item}</p>
										</div>
									</div>
								</GlassCard>
							))}
						</div>

						{state.recentSessions.length > 0 && (
							<div className="mt-6">
								<p className="mb-3 text-sm text-white/50">Recent sessions</p>
								<div className="space-y-2">
									{state.recentSessions.slice(0, 4).map((session) => (
										<div
											key={session.id}
											className="flex items-center justify-between rounded-xl border border-white/10 bg-white/4 px-3 py-2"
										>
											<p className="text-sm text-white/80">Session {session.id.slice(0, 6)}</p>
											<p className="text-sm text-cyan-300">{session.score}/100</p>
										</div>
									))}
								</div>
							</div>
						)}
					</div>
				</GradientBorderCard>
			</div>
		</AppShell>
	);
}
