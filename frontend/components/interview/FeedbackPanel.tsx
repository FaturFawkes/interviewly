import { AlertCircle, CheckCircle, Lightbulb } from "lucide-react";

import { GlassCard, GradientBorderCard } from "@/components/ui/GlassCard";

type FeedbackPanelProps = {
  score: number;
  strengths: string[];
  weaknesses: string[];
  improvements: string[];
  starFeedback: string;
};

export function FeedbackPanel({
  score,
  strengths,
  weaknesses,
  improvements,
  starFeedback,
}: FeedbackPanelProps) {
  const clarity = clampScore(score + 6);
  const relevance = clampScore(score + 2);
  const structure = clampScore(score - 3);
  const depth = clampScore(score - 7);

  return (
    <div className="space-y-5">
      <GradientBorderCard>
        <div className="p-6 text-center">
          <p className="text-white/40 text-sm mb-2">Overall Score</p>
          <div className="relative w-28 h-28 mx-auto mb-3">
            <svg className="w-full h-full -rotate-90" viewBox="0 0 100 100">
              <circle cx="50" cy="50" r="42" fill="none" stroke="rgba(255,255,255,0.08)" strokeWidth="8" />
              <circle
                cx="50"
                cy="50"
                r="42"
                fill="none"
                stroke="url(#scoreGrad)"
                strokeWidth="8"
                strokeLinecap="round"
                strokeDasharray={`${Math.max(0, Math.min(score, 100)) * 2.64} 264`}
              />
              <defs>
                <linearGradient id="scoreGrad" x1="0" y1="0" x2="1" y2="1">
                  <stop offset="0%" stopColor="#8B5CF6" />
                  <stop offset="100%" stopColor="#06B6D4" />
                </linearGradient>
              </defs>
            </svg>
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-3xl text-white">{score}</span>
            </div>
          </div>
          <p className="text-green-400 text-sm">Good performance!</p>
        </div>
      </GradientBorderCard>

      <GlassCard className="p-5">
        <h4 className="text-white text-sm mb-4">Score Breakdown</h4>
        <div className="space-y-3">
          <BreakdownRow label="Clarity" value={clarity} color="from-purple-500 to-purple-400" />
          <BreakdownRow label="Relevance" value={relevance} color="from-blue-500 to-blue-400" />
          <BreakdownRow label="Structure" value={structure} color="from-indigo-500 to-indigo-400" />
          <BreakdownRow label="Depth" value={depth} color="from-cyan-500 to-cyan-400" />
        </div>
      </GlassCard>

      <div className="grid gap-4 md:grid-cols-2">
        <GlassCard className="p-5">
          <div className="flex items-center gap-2 mb-3">
            <CheckCircle className="w-4 h-4 text-green-400" />
            <h4 className="text-white text-sm">Strengths</h4>
          </div>
          <FeedbackList items={strengths} emptyLabel="No strengths yet" iconClassName="text-green-400" />
        </GlassCard>

        <GlassCard className="p-5">
          <div className="flex items-center gap-2 mb-3">
            <AlertCircle className="w-4 h-4 text-yellow-400" />
            <h4 className="text-white text-sm">Areas to Improve</h4>
          </div>
          <FeedbackList items={improvements.length ? improvements : weaknesses} emptyLabel="No improvements yet" iconClassName="text-yellow-400/70" />
        </GlassCard>
      </div>

      <GlassCard className="p-5">
        <div className="flex items-center gap-2 mb-3">
          <Lightbulb className="w-4 h-4 text-cyan-300" />
          <p className="text-xs uppercase tracking-wide text-cyan-200">STAR Guidance</p>
        </div>
        <p className="text-sm leading-relaxed text-white/85">{starFeedback}</p>
      </GlassCard>
    </div>
  );
}

function FeedbackList({
  items,
  emptyLabel,
  iconClassName,
}: {
  items: string[];
  emptyLabel: string;
  iconClassName: string;
}) {
  return (
    <ul className="space-y-2">
      {items.length > 0 ? (
        items.map((item) => (
          <li key={item} className="text-white/50 text-xs flex items-start gap-2">
            <span className={iconClassName}>•</span>
            {item}
          </li>
        ))
      ) : (
        <li className="text-white/35 text-xs">{emptyLabel}</li>
      )}
    </ul>
  );
}

function BreakdownRow({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div>
      <div className="flex justify-between text-xs mb-1">
        <span className="text-white/50">{label}</span>
        <span className="text-white/70">{value}%</span>
      </div>
      <div className="h-1.5 rounded-full bg-white/[0.06]">
        <div className={`h-full rounded-full bg-gradient-to-r ${color}`} style={{ width: `${value}%` }} />
      </div>
    </div>
  );
}

function clampScore(value: number) {
  return Math.max(0, Math.min(100, value));
}
