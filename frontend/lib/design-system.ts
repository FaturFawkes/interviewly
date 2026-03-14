export const designSystem = {
  colors: {
    background: "#0B0F14",
    primaryGradientFrom: "#7B61FF",
    primaryGradientTo: "#2F80ED",
    accentCyan: "#00E5FF",
    textPrimary: "#FFFFFF",
    textSecondary: "#A5B0C2",
    card: "rgba(17, 24, 36, 0.62)",
    border: "rgba(123, 97, 255, 0.32)",
  },
  typography: {
    headingWeight: "600-700",
    bodyWeight: "400-500",
    fontFamily: "Geist, Inter, Satoshi, system-ui, sans-serif",
  },
  spacing: {
    unit: 8,
    scale: [8, 16, 24, 32, 40, 48, 56, 64],
  },
  radius: {
    sm: 16,
    md: 20,
    lg: 24,
  },
  shadows: {
    glow: "0 0 30px rgba(123, 97, 255, 0.28)",
    card: "0 12px 40px rgba(0, 0, 0, 0.35)",
  },
} as const;

export type DesignSystem = typeof designSystem;
