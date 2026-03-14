import { cn } from "@/lib/utils";

type CardProps = React.HTMLAttributes<HTMLDivElement>;

export function Card({ className, ...props }: CardProps) {
  return <div className={cn("glass-card rounded-[20px] p-5", className)} {...props} />;
}
