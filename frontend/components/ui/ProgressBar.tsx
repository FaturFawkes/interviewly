import { cn } from "@/lib/utils";

type ProgressBarProps = {
  value: number;
  className?: string;
};

export function ProgressBar({ value, className }: ProgressBarProps) {
  const normalized = Math.max(0, Math.min(100, value));

  return (
    <div className={cn("h-2 w-full overflow-hidden rounded-full bg-white/10", className)}>
      <div
        className="h-full rounded-full bg-gradient-to-r from-[#7B61FF] to-[#00E5FF] transition-all duration-500"
        style={{ width: `${normalized}%` }}
      />
    </div>
  );
}
