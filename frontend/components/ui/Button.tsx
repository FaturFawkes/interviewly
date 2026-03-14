import { cn } from "@/lib/utils";

type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost";
};

export function Button({ className, variant = "primary", ...props }: ButtonProps) {
  const base =
    "inline-flex items-center justify-center rounded-[16px] px-5 py-2.5 text-sm font-semibold transition duration-200 disabled:cursor-not-allowed disabled:opacity-60";

  const variants = {
    primary:
      "gradient-surface text-white glow-border hover:brightness-110 active:brightness-95 border border-white/10",
    secondary:
      "glass-card border border-[rgba(123,97,255,0.35)] text-white hover:border-cyan-300/70",
    ghost: "text-[var(--color-text-muted)] hover:text-white hover:bg-white/5",
  };

  return <button className={cn(base, variants[variant], className)} {...props} />;
}
