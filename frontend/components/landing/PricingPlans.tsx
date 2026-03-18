"use client";

import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { api } from "@/lib/api/endpoints";
import type { PaymentPlanID } from "@/lib/api/types";
import { cn } from "@/lib/utils";

type PricingPlan = {
  id: PaymentPlanID;
  title: string;
  subtitle: string;
  priceLabel: string;
  badge?: string;
  features: string[];
};

const pricingPlans: PricingPlan[] = [
  {
    id: "starter",
    title: "Starter",
    subtitle: "For focused solo prep",
    priceLabel: "$19",
    features: [
      "30 interview sessions/month",
      "AI scoring + STAR feedback",
      "Progress analytics dashboard",
    ],
  },
  {
    id: "pro",
    title: "Pro Career Boost",
    subtitle: "For high-intent job seekers",
    priceLabel: "$39",
    badge: "Most popular",
    features: [
      "Unlimited sessions",
      "Deep role-fit analysis",
      "Priority AI response speed",
    ],
  },
  {
    id: "elite",
    title: "Elite",
    subtitle: "For accelerated interview mastery",
    priceLabel: "$79",
    features: [
      "Everything in Pro",
      "Advanced interview strategy insights",
      "Priority support",
    ],
  },
];

export function PricingPlans() {
  const [loadingPlanID, setLoadingPlanID] = useState<PaymentPlanID | null>(null);
  const [checkoutError, setCheckoutError] = useState<string | null>(null);

  async function handleCheckout(planID: PaymentPlanID) {
    setCheckoutError(null);
    setLoadingPlanID(planID);

    try {
      const response = await api.createCheckoutSession(planID);
      if (!response.checkout_url) {
        throw new Error("Checkout URL is empty.");
      }

      window.location.href = response.checkout_url;
    } catch (error) {
      setCheckoutError(error instanceof Error ? error.message : "Failed to create checkout session.");
      setLoadingPlanID(null);
    }
  }

  return (
    <section className="space-y-4">
      <div className="text-center">
        <h3 className="text-2xl font-semibold text-white">Simple monthly pricing</h3>
        <p className="mt-2 text-sm text-muted">Choose a plan and continue to secure checkout.</p>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        {pricingPlans.map((plan) => {
          const isLoading = loadingPlanID === plan.id;
          const isHighlighted = plan.id === "pro";

          return (
            <Card key={plan.id} className={cn("rounded-3xl p-6", isHighlighted && "glow-border")}>
              {plan.badge && (
                <p className="inline-flex rounded-full border border-cyan-300/30 bg-cyan-400/10 px-3 py-1 text-xs text-cyan-200">
                  {plan.badge}
                </p>
              )}
              <h3 className={cn("text-xl font-semibold text-white", plan.badge && "mt-3")}>{plan.title}</h3>
              <p className="mt-1 text-sm text-muted">{plan.subtitle}</p>
              <p className="mt-5 text-4xl font-bold text-white">{plan.priceLabel}</p>
              <p className="text-xs text-muted">per month</p>

              <ul className="mt-5 space-y-2 text-sm text-white/90">
                {plan.features.map((feature) => (
                  <li key={feature}>• {feature}</li>
                ))}
              </ul>

              <Button
                className="mt-6 w-full"
                variant={isHighlighted ? "primary" : "secondary"}
                onClick={() => {
                  void handleCheckout(plan.id);
                }}
                disabled={loadingPlanID !== null}
              >
                {isLoading ? "Redirecting..." : "Choose plan"}
              </Button>
            </Card>
          );
        })}
      </div>

      {checkoutError && (
        <p className="text-center text-sm text-red-300">{checkoutError}</p>
      )}
    </section>
  );
}
