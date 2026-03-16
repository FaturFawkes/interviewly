import { Card } from "@/components/ui/Card";

type ChartCardProps = {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
};

export function ChartCard({ title, subtitle, children }: ChartCardProps) {
  return (
    <Card className="rounded-[20px] p-5">
      <div className="mb-4">
        <h3 className="text-base font-semibold text-white">{title}</h3>
        {subtitle && <p className="mt-1 text-sm text-[var(--color-text-muted)]">{subtitle}</p>}
      </div>
      <div className="h-64">{children}</div>
    </Card>
  );
}
