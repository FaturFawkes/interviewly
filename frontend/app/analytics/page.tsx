"use client";

import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { ArrowDownRight, ArrowUpRight, Brain, Target, TrendingUp, Zap } from "lucide-react";

import { AppShell } from "@/components/layout/AppShell";
import { ChartCard } from "@/components/charts/ChartCard";
import { GlassCard, GradientBorderCard } from "@/components/ui/GlassCard";
import { api } from "@/lib/api/endpoints";
import type { AnalyticsPoint } from "@/lib/api/types";

const ScoreHistoryChart = dynamic(
	() => import("@/components/charts/ScoreHistoryChart").then((module) => module.ScoreHistoryChart),
	{ ssr: false },
);

const StrengthWeaknessChart = dynamic(
	() => import("@/components/charts/StrengthWeaknessChart").then((module) => module.StrengthWeaknessChart),
	{ ssr: false },
);

type ProgressState = {
	readiness: number;
	averageScore: number;
	avgScoreTrend: number;
	sessionsCompleted: number;
	practiceStreakDays: number;
	weakAreas: string[];
	recommendations: string[];
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

export default function AnalyticsPage() {
	const [state, setState] = useState<ProgressState>({
		readiness: 0,
		averageScore: 0,
		avgScoreTrend: 0,
		sessionsCompleted: 0,
		practiceStreakDays: 0,
		weakAreas: [],
		recommendations: [],
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
					practiceStreakDays: overview.practice_streak_days,
					weakAreas: overview.weak_areas,
					recommendations: overview.recommendations,
					history: overview.score_history,
					strengthWeakness: buildStrengthWeaknessData(overview.weak_areas, Math.round(overview.average_score)),
				});
			} catch {
				setState((prev) => ({ ...prev, recommendations: [], history: [], strengthWeakness: [] }));
			}
		}

		void load();
	}, []);

	const recommendedImprovements = state.recommendations.length
		? state.recommendations
		: ["No personalized recommendations yet. Complete a practice session first."];

	return (
		<AppShell title="Progress Analytics" subtitle="Track your improvement and identify areas for growth.">
			<div className="space-y-6">
				<section className="grid grid-cols-2 lg:grid-cols-4 gap-4">
					<GlassCard className="p-5" glowColor="purple">
						<div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-purple-500 to-purple-700">
							<Target className="h-5 w-5 text-white" />
						</div>
						<p className="text-white/40 text-sm">Interview Readiness</p>
						<p className="text-2xl text-white mt-0.5">{state.readiness}%</p>
						<p className="text-white/25 text-xs mt-1">Ready for interviews</p>
					</GlassCard>
					<GlassCard className="p-5">
						<div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-green-500 to-green-700">
							<TrendingUp className="h-5 w-5 text-white" />
						</div>
						<p className="text-white/40 text-sm">Avg. Score Trend</p>
						<p className="text-2xl text-white mt-0.5">{state.avgScoreTrend >= 0 ? "+" : ""}{state.avgScoreTrend}</p>
						<p className="text-white/25 text-xs mt-1">Since first session</p>
					</GlassCard>
					<GlassCard className="p-5">
						<div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-blue-500 to-blue-700">
							<Brain className="h-5 w-5 text-white" />
						</div>
						<p className="text-white/40 text-sm">Weak Areas</p>
						<p className="text-2xl text-white mt-0.5">{state.weakAreas.length}</p>
						<p className="text-white/25 text-xs mt-1">Needs practice</p>
					</GlassCard>
					<GlassCard className="p-5">
						<div className="mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-cyan-500 to-cyan-700">
							<Zap className="h-5 w-5 text-white" />
						</div>
						<p className="text-white/40 text-sm">Sessions</p>
						<p className="text-2xl text-white mt-0.5">{state.sessionsCompleted}</p>
						<p className="text-white/25 text-xs mt-1">{state.practiceStreakDays} day streak</p>
					</GlassCard>
				</section>

				<section className="grid gap-4 xl:grid-cols-2">
					<ChartCard title="Score History" subtitle="Behavioral and technical trajectory">
						{state.history.length > 0 ? (
							<ScoreHistoryChart data={state.history} />
						) : (
							<div className="flex h-full items-center justify-center text-sm text-[var(--color-text-muted)]">No session data yet.</div>
						)}
					</ChartCard>
					<ChartCard title="Strength vs Weakness" subtitle="Capability radar summary">
						{state.strengthWeakness.length > 0 ? (
							<StrengthWeaknessChart data={state.strengthWeakness} />
						) : (
							<div className="flex h-full items-center justify-center text-sm text-[var(--color-text-muted)]">No weak-area data yet.</div>
						)}
					</ChartCard>
				</section>

				<GradientBorderCard>
					<div className="p-6">
						<h3 className="text-white mb-1">Practice Recommendations</h3>
						<p className="text-white/40 text-sm mb-5">AI-suggested topics based on your analytics</p>
						<div className="grid md:grid-cols-2 gap-4">
							{recommendedImprovements.map((item, index) => (
								<GlassCard key={`${item}-${index}`} className="p-4 hover:bg-white/[0.04]" glowColor="none">
									<div className="flex items-start gap-3">
										<div className="w-10 h-10 rounded-xl bg-purple-500/10 flex items-center justify-center shrink-0">
											{index % 2 === 0 ? <ArrowUpRight className="w-5 h-5 text-purple-400" /> : <ArrowDownRight className="w-5 h-5 text-cyan-400" />}
										</div>
										<div>
											<p className="text-white/80 text-sm">{item}</p>
										</div>
									</div>
								</GlassCard>
							))}
						</div>
					</div>
				</GradientBorderCard>
			</div>
		</AppShell>
	);
}
