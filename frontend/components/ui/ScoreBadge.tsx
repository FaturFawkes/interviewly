import { cn } from "@/lib/utils";

type ScoreBadgeProps = {
  score: number;
  label?: string;
  className?: string;
};

export function ScoreBadge({ score, label = "Score", className }: ScoreBadgeProps) {
  const tone = score >= 80 ? "text-emerald-300 bg-emerald-400/10 border-emerald-300/40" : score >= 60 ? "text-cyan-300 bg-cyan-400/10 border-cyan-300/40" : "text-amber-300 bg-amber-400/10 border-amber-300/40";

  return (
    <div className={cn("inline-flex items-center gap-2 rounded-full border px-3 py-1", tone, className)}>
      <span className="text-xs font-medium text-white/80">{label}</span>
      <span className="text-sm font-bold">{score}</span>
    </div>
  );
}
