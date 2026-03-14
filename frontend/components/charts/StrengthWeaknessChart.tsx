"use client";

import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

type StrengthWeaknessPoint = {
  area: string;
  strength: number;
  weakness: number;
};

type StrengthWeaknessChartProps = {
  data: StrengthWeaknessPoint[];
};

export function StrengthWeaknessChart({ data }: StrengthWeaknessChartProps) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data} margin={{ top: 6, right: 10, left: 0, bottom: 4 }}>
        <CartesianGrid stroke="rgba(255,255,255,0.08)" vertical={false} />
        <XAxis dataKey="area" stroke="#A5B0C2" tickLine={false} axisLine={false} />
        <YAxis domain={[0, 100]} stroke="#A5B0C2" tickLine={false} axisLine={false} />
        <Tooltip
          contentStyle={{
            background: "rgba(17,24,36,0.95)",
            border: "1px solid rgba(123,97,255,0.35)",
            borderRadius: 14,
          }}
        />
        <Bar dataKey="strength" fill="#2F80ED" radius={[8, 8, 0, 0]} />
        <Bar dataKey="weakness" fill="#7B61FF" radius={[8, 8, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}
