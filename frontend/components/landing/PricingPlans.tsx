"use client";

import { useState } from "react";

import { useLanguage } from "@/components/providers/LanguageProvider";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { api } from "@/lib/api/endpoints";
import { pickLocaleText } from "@/lib/i18n";
import type { PaymentPlanID } from "@/lib/api/types";
import { cn } from "@/lib/utils";

type PricingPlan = {
  id: PaymentPlanID;
  titleID: string;
  titleEN: string;
  subtitleID: string;
  subtitleEN: string;
  priceLabel: string;
  badgeID?: string;
  badgeEN?: string;
  featuresID: string[];
  featuresEN: string[];
};

const pricingPlans: PricingPlan[] = [
  {
    id: "starter",
    titleID: "Starter",
    titleEN: "Starter",
    subtitleID: "Untuk latihan mandiri yang fokus",
    subtitleEN: "For focused solo prep",
    priceLabel: "$19",
    featuresID: [
      "30 sesi interview/bulan",
      "Skoring AI + feedback STAR",
      "Dasbor analitik progres",
    ],
    featuresEN: [
      "30 interview sessions/month",
      "AI scoring + STAR feedback",
      "Progress analytics dashboard",
    ],
  },
  {
    id: "pro",
    titleID: "Pro Career Boost",
    titleEN: "Pro Career Boost",
    subtitleID: "Untuk pencari kerja yang serius",
    subtitleEN: "For high-intent job seekers",
    priceLabel: "$39",
    badgeID: "Paling populer",
    badgeEN: "Most popular",
    featuresID: [
      "Sesi tanpa batas",
      "Analisis kecocokan role mendalam",
      "Kecepatan respons AI prioritas",
    ],
    featuresEN: [
      "Unlimited sessions",
      "Deep role-fit analysis",
      "Priority AI response speed",
    ],
  },
  {
    id: "elite",
    titleID: "Elite",
    titleEN: "Elite",
    subtitleID: "Untuk akselerasi penguasaan interview",
    subtitleEN: "For accelerated interview mastery",
    priceLabel: "$79",
    featuresID: [
      "Semua fitur Pro",
      "Insight strategi interview lanjutan",
      "Dukungan prioritas",
    ],
    featuresEN: [
      "Everything in Pro",
      "Advanced interview strategy insights",
      "Priority support",
    ],
  },
];

export function PricingPlans() {
  const { locale } = useLanguage();
  const [loadingPlanID, setLoadingPlanID] = useState<PaymentPlanID | null>(null);
  const [checkoutError, setCheckoutError] = useState<string | null>(null);

  async function handleCheckout(planID: PaymentPlanID) {
    setCheckoutError(null);
    setLoadingPlanID(planID);

    try {
      const response = await api.createCheckoutSession(planID);
      if (!response.checkout_url) {
        throw new Error(pickLocaleText(locale, "URL checkout kosong.", "Checkout URL is empty."));
      }

      window.location.href = response.checkout_url;
    } catch (error) {
      setCheckoutError(error instanceof Error ? error.message : pickLocaleText(locale, "Gagal membuat sesi checkout.", "Failed to create checkout session."));
      setLoadingPlanID(null);
    }
  }

  return (
    <section className="space-y-4">
      <div className="text-center">
        <h3 className="text-2xl font-semibold text-white">{pickLocaleText(locale, "Paket bulanan sederhana", "Simple monthly pricing")}</h3>
        <p className="mt-2 text-sm text-muted">{pickLocaleText(locale, "Pilih paket lalu lanjut ke checkout aman.", "Choose a plan and continue to secure checkout.")}</p>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        {pricingPlans.map((plan) => {
          const isLoading = loadingPlanID === plan.id;
          const isHighlighted = plan.id === "pro";

          return (
            <Card key={plan.id} className={cn("rounded-3xl p-6", isHighlighted && "glow-border")}>
              {plan.badgeID && (
                <p className="inline-flex rounded-full border border-cyan-300/30 bg-cyan-400/10 px-3 py-1 text-xs text-cyan-200">
                  {pickLocaleText(locale, plan.badgeID ?? "", plan.badgeEN ?? "")}
                </p>
              )}
              <h3 className={cn("text-xl font-semibold text-white", plan.badgeID && "mt-3")}>{pickLocaleText(locale, plan.titleID, plan.titleEN)}</h3>
              <p className="mt-1 text-sm text-muted">{pickLocaleText(locale, plan.subtitleID, plan.subtitleEN)}</p>
              <p className="mt-5 text-4xl font-bold text-white">{plan.priceLabel}</p>
              <p className="text-xs text-muted">{pickLocaleText(locale, "per bulan", "per month")}</p>

              <ul className="mt-5 space-y-2 text-sm text-white/90">
                {(locale === "id" ? plan.featuresID : plan.featuresEN).map((feature) => (
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
                {isLoading ? pickLocaleText(locale, "Mengalihkan...", "Redirecting...") : pickLocaleText(locale, "Pilih paket", "Choose plan")}
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
