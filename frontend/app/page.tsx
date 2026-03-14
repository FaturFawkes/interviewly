import Link from "next/link";
import { ArrowRight, BarChart3, Brain, Check, FileSearch, MessageSquare, Shield, Sparkles, Star, Zap } from "lucide-react";

import { Navbar } from "@/components/layout/Navbar";
import { GlassCard, GradientBorderCard } from "@/components/ui/GlassCard";
import { GradientButton } from "@/components/ui/GradientButton";
import { testimonials } from "@/lib/mock-data";

const featureItems = [
  {
    icon: MessageSquare,
    title: "AI Mock Interviews",
    description: "Practice with an AI interviewer that adapts to your role and experience level.",
    color: "from-purple-500 to-purple-700",
  },
  {
    icon: FileSearch,
    title: "JD Analysis",
    description: "Upload any job description and get targeted interview questions instantly.",
    color: "from-blue-500 to-blue-700",
  },
  {
    icon: BarChart3,
    title: "Feedback Scoring",
    description: "Get detailed scoring on communication, technical depth, and confidence.",
    color: "from-cyan-500 to-cyan-700",
  },
  {
    icon: Brain,
    title: "Smart Prep Plans",
    description: "AI-generated practice plans based on your strengths and weaknesses.",
    color: "from-violet-500 to-violet-700",
  },
  {
    icon: Zap,
    title: "Real-time Feedback",
    description: "Instant suggestions to improve your answers as you practice.",
    color: "from-indigo-500 to-indigo-700",
  },
  {
    icon: Shield,
    title: "Industry Coverage",
    description: "From tech to finance — questions tailored to your target industry.",
    color: "from-teal-500 to-teal-700",
  },
];

const pricingPlans = [
  {
    name: "Free",
    price: "$0",
    period: "forever",
    features: ["3 mock interviews/month", "Basic feedback", "1 JD analysis", "Community support"],
    highlighted: false,
  },
  {
    name: "Pro",
    price: "$19",
    period: "/month",
    features: ["Unlimited mock interviews", "Detailed AI feedback", "Unlimited JD analysis", "Progress analytics", "Priority support", "Custom practice plans"],
    highlighted: true,
  },
  {
    name: "Enterprise",
    price: "$49",
    period: "/month",
    features: ["Everything in Pro", "Team management", "Bulk interview sessions", "API access", "Dedicated account manager", "Custom integrations"],
    highlighted: false,
  },
];

export default function Home() {
  return (
    <div className="min-h-screen bg-[#0B0F14] text-white overflow-x-hidden relative">
      <div className="fixed inset-0 pointer-events-none">
        <div className="absolute top-[-20%] left-[-10%] w-[600px] h-[600px] rounded-full bg-purple-600/[0.07] blur-[120px]" />
        <div className="absolute top-[30%] right-[-15%] w-[500px] h-[500px] rounded-full bg-cyan-500/[0.05] blur-[120px]" />
        <div className="absolute bottom-[-10%] left-[30%] w-[400px] h-[400px] rounded-full bg-blue-600/[0.06] blur-[120px]" />
      </div>

      <Navbar />

      <main className="relative z-10">
        <section className="max-w-7xl mx-auto px-8 pt-20 pb-32 text-center">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full border border-purple-500/20 bg-purple-500/[0.08] text-purple-300 text-sm mb-8">
            <Sparkles className="w-3.5 h-3.5" />
            Powered by GPT-4 & advanced AI
          </div>
          <h1 className="text-5xl md:text-7xl tracking-tight mb-6 bg-gradient-to-br from-white via-white/90 to-white/50 bg-clip-text text-transparent max-w-4xl mx-auto leading-[1.1]">
            Ace Your Next Interview with AI
          </h1>
          <p className="text-lg text-white/50 max-w-2xl mx-auto mb-10 leading-relaxed">
            Practice with an AI interviewer, get instant feedback, and track your progress.
            Land your dream job with confidence.
          </p>
          <div className="flex items-center justify-center gap-4">
            <Link href="/dashboard">
              <GradientButton size="lg">
                Start Practicing Interviews
                <ArrowRight className="w-5 h-5 ml-2 inline" />
              </GradientButton>
            </Link>
            <GradientButton variant="outline" size="lg">Watch Demo</GradientButton>
          </div>

          <div className="mt-20">
            <GradientBorderCard className="max-w-5xl mx-auto">
              <div className="p-8 rounded-2xl">
                <div className="bg-[#080B10] rounded-xl p-6 border border-white/[0.04]">
                  <div className="flex items-center gap-2 mb-6">
                    <div className="w-3 h-3 rounded-full bg-red-500/60" />
                    <div className="w-3 h-3 rounded-full bg-yellow-500/60" />
                    <div className="w-3 h-3 rounded-full bg-green-500/60" />
                    <span className="text-white/30 text-xs ml-2">AI Interview Coach — Mock Session</span>
                  </div>
                  <div className="grid md:grid-cols-2 gap-6">
                    <div className="space-y-4">
                      <div className="p-4 rounded-xl bg-purple-500/[0.08] border border-purple-500/20 text-left">
                        <p className="text-sm text-purple-300 mb-1">AI Interviewer</p>
                        <p className="text-white/80 text-sm">Tell me about a time you had to lead a project under tight deadlines. How did you handle it?</p>
                      </div>
                      <div className="p-4 rounded-xl bg-white/[0.03] border border-white/[0.06] text-left">
                        <p className="text-sm text-cyan-300 mb-1">Your Answer</p>
                        <p className="text-white/60 text-sm">In my previous role at Acme Corp, I led a 5-person team to deliver a critical feature...</p>
                      </div>
                    </div>
                    <div className="space-y-4 text-left">
                      <div className="p-4 rounded-xl bg-white/[0.03] border border-white/[0.06]">
                        <p className="text-sm text-cyan-400 mb-3">Live Feedback</p>
                        <div className="space-y-3">
                          <MetricBar label="Clarity" value="92%" width="w-[92%]" color="from-purple-500 to-cyan-400" />
                          <MetricBar label="Relevance" value="87%" width="w-[87%]" color="from-blue-500 to-purple-500" />
                          <MetricBar label="Depth" value="78%" width="w-[78%]" color="from-cyan-500 to-blue-500" />
                        </div>
                      </div>
                      <div className="flex items-center gap-3 p-3 rounded-xl bg-green-500/[0.08] border border-green-500/20">
                        <div className="w-10 h-10 rounded-lg bg-green-500/20 flex items-center justify-center">
                          <BarChart3 className="w-5 h-5 text-green-400" />
                        </div>
                        <div>
                          <p className="text-green-400 text-sm">Overall Score</p>
                          <p className="text-white text-lg">86/100</p>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </GradientBorderCard>
          </div>
        </section>

        <section id="features" className="relative z-10 max-w-7xl mx-auto px-8 py-24">
          <div className="text-center mb-16">
            <p className="text-purple-400 text-sm mb-3 tracking-wide uppercase">Features</p>
            <h2 className="text-3xl md:text-4xl tracking-tight mb-4">Everything you need to ace your interview</h2>
            <p className="text-white/40 max-w-xl mx-auto">Comprehensive AI-powered tools to prepare, practice, and perfect your interview skills.</p>
          </div>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-5">
            {featureItems.map((feature, i) => (
              <GlassCard glowColor={i % 3 === 0 ? "purple" : i % 3 === 1 ? "blue" : "cyan"} className="p-6 h-full" key={feature.title}>
                <div className={`w-11 h-11 rounded-xl bg-gradient-to-br ${feature.color} flex items-center justify-center mb-4`}>
                  <feature.icon className="w-5 h-5 text-white" />
                </div>
                <h3 className="text-white mb-2">{feature.title}</h3>
                <p className="text-white/40 text-sm leading-relaxed">{feature.description}</p>
              </GlassCard>
            ))}
          </div>
        </section>

        <section id="testimonials" className="relative z-10 max-w-7xl mx-auto px-8 py-24">
          <div className="text-center mb-16">
            <p className="text-cyan-400 text-sm mb-3 tracking-wide uppercase">Testimonials</p>
            <h2 className="text-3xl md:text-4xl tracking-tight mb-4">Loved by thousands of job seekers</h2>
          </div>
          <div className="grid md:grid-cols-3 gap-5">
            {testimonials.slice(0, 3).map((item) => (
              <GlassCard key={item.name} className="p-6 h-full">
                <div className="flex gap-1 mb-4">
                  {Array.from({ length: 5 }).map((_, index) => (
                    <Star key={`${item.name}-${index}`} className="w-4 h-4 text-yellow-400 fill-yellow-400" />
                  ))}
                </div>
                <p className="text-white/70 text-sm mb-6 leading-relaxed">&ldquo;{item.quote}&rdquo;</p>
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-gradient-to-br from-purple-500/50 to-cyan-500/50 flex items-center justify-center text-sm text-white">
                    {item.name.split(" ").map((value) => value[0]).join("")}
                  </div>
                  <div>
                    <p className="text-white text-sm">{item.name}</p>
                    <p className="text-white/40 text-xs">{item.role}</p>
                  </div>
                </div>
              </GlassCard>
            ))}
          </div>
        </section>

        <section id="pricing" className="relative z-10 max-w-7xl mx-auto px-8 py-24">
          <div className="text-center mb-16">
            <p className="text-purple-400 text-sm mb-3 tracking-wide uppercase">Pricing</p>
            <h2 className="text-3xl md:text-4xl tracking-tight mb-4">Simple, transparent pricing</h2>
            <p className="text-white/40 max-w-xl mx-auto">Start free, upgrade when you&apos;re ready. No hidden fees.</p>
          </div>
          <div className="grid md:grid-cols-3 gap-6 max-w-5xl mx-auto">
            {pricingPlans.map((plan) => (
              <div key={plan.name}>
                {plan.highlighted ? (
                  <GradientBorderCard>
                    <div className="p-7">
                      <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-purple-500/15 text-purple-300 text-xs mb-4">
                        <Sparkles className="w-3 h-3" /> Most Popular
                      </div>
                      <h3 className="text-white text-xl mb-1">{plan.name}</h3>
                      <div className="flex items-baseline gap-1 mb-6">
                        <span className="text-4xl text-white">{plan.price}</span>
                        <span className="text-white/40 text-sm">{plan.period}</span>
                      </div>
                      <Link href="/dashboard">
                        <GradientButton fullWidth className="mb-6">Get Started</GradientButton>
                      </Link>
                      <ul className="space-y-3">
                        {plan.features.map((feature) => (
                          <li key={feature} className="flex items-center gap-2.5 text-sm text-white/60">
                            <Check className="w-4 h-4 text-purple-400 shrink-0" />
                            {feature}
                          </li>
                        ))}
                      </ul>
                    </div>
                  </GradientBorderCard>
                ) : (
                  <GlassCard className="p-7 h-full">
                    <h3 className="text-white text-xl mb-1">{plan.name}</h3>
                    <div className="flex items-baseline gap-1 mb-6">
                      <span className="text-4xl text-white">{plan.price}</span>
                      <span className="text-white/40 text-sm">{plan.period}</span>
                    </div>
                    <Link href="/dashboard">
                      <GradientButton variant="outline" fullWidth className="mb-6">Get Started</GradientButton>
                    </Link>
                    <ul className="space-y-3">
                      {plan.features.map((feature) => (
                        <li key={feature} className="flex items-center gap-2.5 text-sm text-white/60">
                          <Check className="w-4 h-4 text-white/30 shrink-0" />
                          {feature}
                        </li>
                      ))}
                    </ul>
                  </GlassCard>
                )}
              </div>
            ))}
          </div>
        </section>
      </main>

      <footer className="relative z-10 border-t border-white/[0.06] mt-12">
        <div className="max-w-7xl mx-auto px-8 py-12 flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-purple-500 to-cyan-400 flex items-center justify-center">
              <Sparkles className="w-4 h-4 text-white" />
            </div>
            <span className="text-white/60 text-sm">AI Interview Coach</span>
          </div>
          <p className="text-white/30 text-sm">&copy; 2026 AI Interview Coach. All rights reserved.</p>
        </div>
      </footer>
    </div>
  );
}

function MetricBar({
  label,
  value,
  width,
  color,
}: {
  label: string;
  value: string;
  width: string;
  color: string;
}) {
  return (
    <div>
      <div className="flex justify-between text-xs text-white/50 mb-1">
        <span>{label}</span>
        <span>{value}</span>
      </div>
      <div className="h-1.5 rounded-full bg-white/[0.06]">
        <div className={`h-full ${width} rounded-full bg-gradient-to-r ${color}`} />
      </div>
    </div>
  );
}
