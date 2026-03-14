import { Mic, Timer } from "lucide-react";

import { Card } from "@/components/ui/Card";
import { ScoreBadge } from "@/components/ui/ScoreBadge";

type InterviewPanelProps = {
  question: string;
  type: string;
  timerSeconds: number;
  current: number;
  total: number;
  currentScore: number;
};

export function InterviewPanel({
  question,
  type,
  timerSeconds,
  current,
  total,
  currentScore,
}: InterviewPanelProps) {
  const minutes = Math.floor(timerSeconds / 60);
  const seconds = timerSeconds % 60;

  return (
    <Card className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2 rounded-full border border-cyan-300/30 bg-cyan-400/10 px-3 py-1 text-xs font-medium text-cyan-200">
          <Mic className="h-3.5 w-3.5" />
          AI Interviewer · {type}
        </div>

        <div className="flex items-center gap-2">
          <div className="inline-flex items-center gap-1 rounded-full border border-white/15 px-3 py-1 text-xs text-white/80">
            <Timer className="h-3.5 w-3.5" />
            {minutes.toString().padStart(2, "0")}:{seconds.toString().padStart(2, "0")}
          </div>
          <ScoreBadge score={currentScore} label="Latest" />
        </div>
      </div>

      <div>
        <p className="mb-2 text-xs uppercase tracking-wide text-[var(--color-text-muted)]">
          Question {current} of {total}
        </p>
        <h2 className="text-lg font-semibold leading-relaxed text-white">{question}</h2>
      </div>
    </Card>
  );
}
