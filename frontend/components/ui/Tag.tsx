import { cn } from "@/lib/utils";

type TagProps = {
  children: React.ReactNode;
  className?: string;
};

export function Tag({ children, className }: TagProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border border-cyan-300/35 bg-cyan-400/10 px-3 py-1 text-xs font-medium text-cyan-200",
        className,
      )}
    >
      {children}
    </span>
  );
}
