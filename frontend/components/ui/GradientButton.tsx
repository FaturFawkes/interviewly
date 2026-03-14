import { cn } from "@/lib/utils";

type GradientButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "solid" | "outline";
  size?: "sm" | "md" | "lg";
  fullWidth?: boolean;
};

export function GradientButton({
  className,
  variant = "solid",
  size = "md",
  fullWidth = false,
  ...props
}: GradientButtonProps) {
  const sizeClass = {
    sm: "h-9 px-4 text-sm",
    md: "h-10 px-5 text-sm",
    lg: "h-12 px-6 text-base",
  }[size];

  const variantClass =
    variant === "solid"
      ? "bg-gradient-to-r from-purple-500 to-cyan-500 text-white border border-white/10 shadow-[0_10px_30px_rgba(123,97,255,0.25)] hover:brightness-110"
      : "bg-transparent text-white/85 border border-white/20 hover:bg-white/[0.04]";

  return (
    <button
      className={cn(
        "inline-flex items-center justify-center rounded-xl font-medium transition-all duration-200 disabled:opacity-60 disabled:cursor-not-allowed",
        sizeClass,
        variantClass,
        fullWidth && "w-full",
        className,
      )}
      {...props}
    />
  );
}
