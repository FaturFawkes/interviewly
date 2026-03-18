import Link from "next/link";
import { ArrowRight, Brain, ChartNoAxesCombined, FileText, Sparkles } from "lucide-react";

import { Navbar } from "@/components/layout/Navbar";
import { PricingPlans } from "@/components/landing/PricingPlans";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { testimonials } from "@/lib/mock-data";

const featureItems = [
  {
    title: "AI Mock Interviews",
    desc: "Practice realistic interview questions personalized to your resume and target role.",
    icon: Brain,
  },
  {
    title: "JD Analysis",
    desc: "Extract skills, keywords, and themes from job descriptions to align answers better.",
    icon: FileText,
  },
  {
    title: "Feedback Scoring",
    desc: "Get instant scoring, strengths, weaknesses, and STAR improvements after every answer.",
    icon: ChartNoAxesCombined,
  },
];

export default function Home() {
  return (
    <div className="relative min-h-screen overflow-hidden pb-10">
      <div className="ambient-orb orb-primary -left-20 top-6 h-64 w-64" />
      <div className="ambient-orb orb-cyan right-0 top-44 h-72 w-72" />

      <Navbar />

      <main className="section-shell relative z-10 space-y-16 pb-10 pt-8 sm:space-y-20 sm:pt-12">
        <section className="grid items-center gap-10 lg:grid-cols-2">
          <div className="space-y-6">
            <p className="inline-flex items-center gap-2 rounded-full border border-white/15 px-3 py-1 text-xs font-semibold text-cyan-200">
              <Sparkles className="h-3.5 w-3.5" />
              AI-powered career acceleration
            </p>

            <h1 className="text-4xl font-bold leading-tight tracking-tight text-white sm:text-5xl lg:text-6xl">
              Practice interviews with your <span className="gradient-text">AI Interview Coach</span>
            </h1>

            <p className="max-w-xl text-base leading-relaxed text-[var(--color-text-muted)] sm:text-lg">
              Turn any job description into a focused mock interview flow, receive instant coaching, and track
              readiness with premium analytics.
            </p>

            <div className="flex flex-wrap items-center gap-3">
              <Link href="/practice">
                <Button className="gap-2">
                  Start practicing interviews
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </Link>
              <Link href="/dashboard">
                <Button variant="secondary">View dashboard</Button>
              </Link>
            </div>
          </div>

          <Card className="relative min-h-[320px] overflow-hidden rounded-[24px] p-6">
            <div className="absolute inset-0 grid-overlay opacity-25" />
            <div className="relative space-y-4">
              <p className="text-sm text-[var(--color-text-muted)]">Live product preview</p>
              <div className="grid gap-3">
                <PreviewRow title="Interview readiness" value="82%" />
                <PreviewRow title="Latest score" value="88" />
                <PreviewRow title="Weak area" value="STAR structure" />
              </div>
              <div className="mt-4 rounded-[16px] border border-cyan-300/30 bg-cyan-400/10 p-4 text-sm text-cyan-100">
                AI suggests: “Anchor answer with measurable impact and close with role relevance.”
              </div>
            </div>
          </Card>
        </section>

        <section className="grid gap-4 md:grid-cols-3">
          {featureItems.map(({ title, desc, icon: Icon }) => (
            <Card key={title} className="rounded-[20px] p-5">
              <div className="mb-4 inline-flex rounded-[14px] border border-cyan-300/35 bg-cyan-400/10 p-2 text-cyan-200">
                <Icon className="h-4 w-4" />
              </div>
              <h3 className="text-lg font-semibold text-white">{title}</h3>
              <p className="mt-2 text-sm leading-relaxed text-[var(--color-text-muted)]">{desc}</p>
            </Card>
          ))}
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          {[
            ["1", "Paste resume + JD", "Upload your target role context and experience."],
            ["2", "Generate interview", "AI creates tailored technical and behavioral questions."],
            ["3", "Get scored feedback", "Improve with actionable STAR-based coaching."],
          ].map(([step, title, desc]) => (
            <Card key={title} className="rounded-[20px] p-5">
              <p className="text-sm font-semibold text-cyan-300">Step {step}</p>
              <h3 className="mt-2 text-lg font-semibold text-white">{title}</h3>
              <p className="mt-2 text-sm text-[var(--color-text-muted)]">{desc}</p>
            </Card>
          ))}
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          {testimonials.map((item) => (
            <Card key={item.name} className="rounded-[20px] p-5">
              <p className="text-sm leading-relaxed text-white/90">“{item.quote}”</p>
              <div className="mt-4">
                <p className="text-sm font-semibold text-white">{item.name}</p>
                <p className="text-xs text-[var(--color-text-muted)]">{item.role}</p>
              </div>
            </Card>
          ))}
        </section>

        <PricingPlans />
      </main>

      <footer className="section-shell relative z-10 border-t border-white/10 pt-7 pb-5 text-sm text-[var(--color-text-muted)]">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <p>© 2026 AI Interview Coach</p>
          <p>Built for modern job seekers.</p>
        </div>
      </footer>
    </div>
  );
}

function PreviewRow({ title, value }: { title: string; value: string }) {
  return (
    <div className="flex items-center justify-between rounded-[16px] border border-white/10 bg-white/5 px-4 py-3">
      <p className="text-sm text-[var(--color-text-muted)]">{title}</p>
      <p className="text-sm font-semibold text-white">{value}</p>
    </div>
  );
}
