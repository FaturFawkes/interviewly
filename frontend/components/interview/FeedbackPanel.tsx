import { Card } from "@/components/ui/Card";
import { ScoreBadge } from "@/components/ui/ScoreBadge";

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
  return (
    <Card className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-base font-semibold text-white">AI Feedback</h3>
        <ScoreBadge score={score} />
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <FeedbackList title="Strengths" items={strengths} />
        <FeedbackList title="Weaknesses" items={weaknesses} />
        <FeedbackList title="Improvements" items={improvements} />
      </div>

      <div className="rounded-[14px] border border-cyan-300/30 bg-cyan-400/10 p-3">
        <p className="text-xs uppercase tracking-wide text-cyan-200">STAR Guidance</p>
        <p className="mt-1 text-sm leading-relaxed text-white/90">{starFeedback}</p>
      </div>
    </Card>
  );
}

function FeedbackList({ title, items }: { title: string; items: string[] }) {
  return (
    <div>
      <p className="mb-2 text-xs uppercase tracking-wide text-[var(--color-text-muted)]">{title}</p>
      <ul className="space-y-1.5 text-sm text-white/90">
        {items.length > 0 ? (
          items.map((item) => <li key={item}>• {item}</li>)
        ) : (
          <li className="text-[var(--color-text-muted)]">No data yet</li>
        )}
      </ul>
    </div>
  );
}
