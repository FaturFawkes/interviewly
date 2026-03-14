"use client";

import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

type ScorePoint = {
  label: string;
  score: number;
};

type ScoreHistoryChartProps = {
  data: ScorePoint[];
};

export function ScoreHistoryChart({ data }: ScoreHistoryChartProps) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <LineChart data={data} margin={{ top: 12, right: 14, left: 0, bottom: 4 }}>
        <XAxis dataKey="label" stroke="#A5B0C2" tickLine={false} axisLine={false} />
        <YAxis domain={[0, 100]} stroke="#A5B0C2" tickLine={false} axisLine={false} />
        <Tooltip
          contentStyle={{
            background: "rgba(17,24,36,0.95)",
            border: "1px solid rgba(123,97,255,0.35)",
            borderRadius: 14,
          }}
        />
        <Line
          dataKey="score"
          type="monotone"
          stroke="url(#scoreGradient)"
          strokeWidth={3}
          dot={{ r: 4, fill: "#00E5FF" }}
          activeDot={{ r: 6, fill: "#7B61FF" }}
        />
        <defs>
          <linearGradient id="scoreGradient" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stopColor="#7B61FF" />
            <stop offset="100%" stopColor="#00E5FF" />
          </linearGradient>
        </defs>
      </LineChart>
    </ResponsiveContainer>
  );
}
