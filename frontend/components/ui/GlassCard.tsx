import { cn } from "@/lib/utils";

type GlowColor = "none" | "purple" | "cyan" | "blue";

type GlassCardProps = React.HTMLAttributes<HTMLDivElement> & {
  glowColor?: GlowColor;
};

const glowClassByColor: Record<GlowColor, string> = {
  none: "",
  purple: "shadow-[0_0_40px_rgba(123,97,255,0.22)]",
  cyan: "shadow-[0_0_40px_rgba(0,229,255,0.16)]",
  blue: "shadow-[0_0_40px_rgba(47,128,237,0.2)]",
};

export function GlassCard({ className, glowColor = "none", ...props }: GlassCardProps) {
  return (
    <div
      className={cn(
        "glass-card rounded-[20px] border border-white/10 bg-[rgba(17,24,36,0.62)] backdrop-blur-md",
        glowClassByColor[glowColor],
        className,
      )}
      {...props}
    />
  );
}

export function GradientBorderCard({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "rounded-[22px] bg-linear-to-br from-[rgba(123,97,255,0.45)] via-[rgba(47,128,237,0.35)] to-[rgba(0,229,255,0.28)] p-px",
        className,
      )}
    >
      <div className="glass-card rounded-[21px] border border-white/10 bg-[rgba(17,24,36,0.62)] backdrop-blur-md" {...props} />
    </div>
  );
}
