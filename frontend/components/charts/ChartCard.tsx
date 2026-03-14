import { GlassCard } from "@/components/ui/GlassCard";

type ChartCardProps = {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
};

export function ChartCard({ title, subtitle, children }: ChartCardProps) {
  return (
    <GlassCard className="p-6">
      <div className="mb-5">
        <h3 className="text-white">{title}</h3>
        {subtitle && <p className="mt-0.5 text-sm text-white/40">{subtitle}</p>}
      </div>
      <div className="h-[240px]">{children}</div>
    </GlassCard>
  );
}
