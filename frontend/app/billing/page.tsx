"use client";

import { useEffect, useMemo, useState } from "react";
import { ArrowRight, Clock3, MicVocal, ReceiptText, Sparkles } from "lucide-react";

import { AppShell } from "@/components/layout/AppShell";
import { useLanguage } from "@/components/providers/LanguageProvider";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { api } from "@/lib/api/endpoints";
import { pickLocaleText } from "@/lib/i18n";
import type { PaymentPlanID, SubscriptionStatus, VoiceTopupPackageCode } from "@/lib/api/types";

type VoiceTopupCard = {
  code: VoiceTopupPackageCode;
  minutes: number;
  priceID: string;
  priceEN: string;
  descriptionID: string;
  descriptionEN: string;
};

type PlanCard = {
  planID: PaymentPlanID;
  titleID: string;
  titleEN: string;
  priceID: string;
  priceEN: string;
  subtitleID: string;
  subtitleEN: string;
};

const voiceTopupCards: VoiceTopupCard[] = [
  {
    code: "voice_topup_10",
    minutes: 10,
    priceID: "Rp19.000",
    priceEN: "IDR 19,000",
    descriptionID: "Top-up cepat untuk lanjutkan sesi voice saat kuota bulanan habis.",
    descriptionEN: "Quick top-up to continue voice sessions when monthly quota runs out.",
  },
  {
    code: "voice_topup_30",
    minutes: 30,
    priceID: "Rp49.000",
    priceEN: "IDR 49,000",
    descriptionID: "Paket hemat dengan menit lebih besar untuk latihan intensif.",
    descriptionEN: "Better value package with more minutes for intensive practice.",
  },
];

const planCards: PlanCard[] = [
  {
    planID: "starter",
    titleID: "Starter",
    titleEN: "Starter",
    priceID: "Rp59.000",
    priceEN: "IDR 59,000",
    subtitleID: "Untuk latihan rutin yang fokus.",
    subtitleEN: "For focused routine interview prep.",
  },
  {
    planID: "pro",
    titleID: "Pro Career Boost",
    titleEN: "Pro Career Boost",
    priceID: "Rp129.000",
    priceEN: "IDR 129,000",
    subtitleID: "Unlimited text + prioritas AI response.",
    subtitleEN: "Unlimited text + priority AI response.",
  },
  {
    planID: "elite",
    titleID: "Elite",
    titleEN: "Elite",
    priceID: "Rp279.000",
    priceEN: "IDR 279,000",
    subtitleID: "Kapasitas voice terbesar untuk sesi intensif.",
    subtitleEN: "Highest voice capacity for intensive sessions.",
  },
];

export default function BillingPage() {
  const { locale } = useLanguage();
  const [status, setStatus] = useState<SubscriptionStatus | null>(null);
  const [statusError, setStatusError] = useState<string | null>(null);
  const [checkoutError, setCheckoutError] = useState<string | null>(null);
  const [loadingKey, setLoadingKey] = useState<string | null>(null);

  useEffect(() => {
    const loadStatus = async () => {
      setStatusError(null);
      try {
        const response = await api.getSubscriptionStatus();
        setStatus(response);
      } catch (error) {
        setStatusError(error instanceof Error ? error.message : pickLocaleText(locale, "Gagal memuat status billing.", "Failed to load billing status."));
      }
    };

    void loadStatus();
  }, [locale]);

  const currentPlanLabel = useMemo(() => {
    const planID = (status?.plan_id ?? "free").toLowerCase();
    if (planID === "elite") {
      return "Elite";
    }
    if (planID === "pro") {
      return "Pro Career Boost";
    }
    if (planID === "starter") {
      return "Starter";
    }
    return pickLocaleText(locale, "Free", "Free");
  }, [locale, status?.plan_id]);

  const overallVoiceRemaining = status?.remaining_voice_minutes ?? 0;
  const topupRemaining = status?.remaining_voice_topup_minutes ?? 0;
  const baseQuotaRemaining = Math.max(overallVoiceRemaining - topupRemaining, 0);
  const remainingSessionsValue = status?.remaining_sessions ?? 0;
  const remainingSessionsLabel = remainingSessionsValue < 0
    ? pickLocaleText(locale, "Tanpa batas", "Unlimited")
    : `${remainingSessionsValue}`;

  async function handleTopupCheckout(packageCode: VoiceTopupPackageCode): Promise<void> {
    setCheckoutError(null);
    setLoadingKey(packageCode);

    try {
      const response = await api.createVoiceTopupCheckoutSession(packageCode);
      if (!response.checkout_url) {
        throw new Error(pickLocaleText(locale, "URL checkout top-up kosong.", "Top-up checkout URL is empty."));
      }

      window.location.href = response.checkout_url;
    } catch (error) {
      setCheckoutError(error instanceof Error ? error.message : pickLocaleText(locale, "Gagal membuat checkout top-up.", "Failed to create top-up checkout."));
      setLoadingKey(null);
    }
  }

  async function handlePlanCheckout(planID: PaymentPlanID): Promise<void> {
    setCheckoutError(null);
    setLoadingKey(planID);

    try {
      const response = await api.createCheckoutSession(planID);
      if (!response.checkout_url) {
        throw new Error(pickLocaleText(locale, "URL checkout paket kosong.", "Plan checkout URL is empty."));
      }

      window.location.href = response.checkout_url;
    } catch (error) {
      setCheckoutError(error instanceof Error ? error.message : pickLocaleText(locale, "Gagal membuat checkout paket.", "Failed to create plan checkout."));
      setLoadingKey(null);
    }
  }

  return (
    <AppShell
      title={pickLocaleText(locale, "Billing & Top-Up", "Billing & Top-Up")}
      subtitle={pickLocaleText(locale, "Kelola paket berlangganan dan tambahkan menit voice kapan saja.", "Manage your subscription plans and add voice minutes anytime.")}
    >
      <div className="space-y-5">
        <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Paket aktif", "Current plan")}</p>
            <p className="mt-2 text-2xl font-bold text-white">{currentPlanLabel}</p>
            <p className="mt-3 text-sm text-white/75">{pickLocaleText(locale, "Upgrade kapan saja untuk kuota dan prioritas lebih tinggi.", "Upgrade anytime for higher quota and priority.")}</p>
          </Card>

          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Sisa voice total", "Total voice remaining")}</p>
            <p className="mt-2 text-2xl font-bold text-white">{overallVoiceRemaining} {pickLocaleText(locale, "menit", "min")}</p>
            <p className="mt-3 text-sm text-white/75">{pickLocaleText(locale, `Kuota dasar tersisa ${baseQuotaRemaining} menit.`, `${baseQuotaRemaining} min remaining from base quota.`)}</p>
          </Card>

          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Saldo top-up voice", "Voice top-up balance")}</p>
            <p className="mt-2 text-2xl font-bold text-white">{topupRemaining} {pickLocaleText(locale, "menit", "min")}</p>
            <p className="mt-3 text-sm text-white/75">{pickLocaleText(locale, "Saldo top-up akan carry-over sampai habis.", "Top-up balance carries over until exhausted.")}</p>
          </Card>

          <Card>
            <p className="text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Permintaan AI tersisa", "Remaining AI requests")}</p>
            <p className="mt-2 text-2xl font-bold text-white">{status?.remaining_text_requests ?? 0}</p>
            <p className="mt-3 text-sm text-white/75">
              {status?.text_fup_exceeded
                ? pickLocaleText(locale, "FUP terlampaui: response bisa lebih lambat.", "FUP exceeded: responses may be slowed down.")
                : pickLocaleText(locale, "Masih dalam batas fair usage period ini.", "Still within fair usage for this period.")}
            </p>
          </Card>
        </section>

        {statusError && (
          <Card className="border border-red-400/30 bg-red-500/10">
            <p className="text-sm text-red-200">{statusError}</p>
          </Card>
        )}

        <section className="grid gap-4 xl:grid-cols-2">
          <Card>
            <div className="mb-4 flex items-center gap-2">
              <MicVocal className="h-4 w-4 text-cyan-300" />
              <h3 className="text-base font-semibold text-white">{pickLocaleText(locale, "Voice Top-Up", "Voice Top-Up")}</h3>
            </div>

            <div className="space-y-3">
              {voiceTopupCards.map((item) => {
                const loading = loadingKey === item.code;
                return (
                  <div key={item.code} className="rounded-[16px] border border-white/10 bg-white/5 p-4">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div>
                        <p className="text-sm font-semibold text-white">{item.minutes} {pickLocaleText(locale, "Menit Voice", "Voice Minutes")}</p>
                        <p className="text-xs text-white/60">{pickLocaleText(locale, item.descriptionID, item.descriptionEN)}</p>
                      </div>
                      <p className="text-lg font-bold text-cyan-100">{pickLocaleText(locale, item.priceID, item.priceEN)}</p>
                    </div>

                    <Button
                      className="mt-3 w-full"
                      onClick={() => {
                        void handleTopupCheckout(item.code);
                      }}
                      disabled={loadingKey !== null}
                    >
                      {loading ? pickLocaleText(locale, "Mengalihkan ke checkout...", "Redirecting to checkout...") : pickLocaleText(locale, "Beli top-up", "Buy top-up")}
                      {!loading && <ArrowRight className="ml-2 h-4 w-4" />}
                    </Button>
                  </div>
                );
              })}
            </div>
          </Card>

          <Card>
            <div className="mb-4 flex items-center gap-2">
              <Sparkles className="h-4 w-4 text-purple-300" />
              <h3 className="text-base font-semibold text-white">{pickLocaleText(locale, "Upgrade Paket", "Plan Upgrade")}</h3>
            </div>

            <div className="space-y-3">
              {planCards.map((item) => {
                const loading = loadingKey === item.planID;
                return (
                  <div key={item.planID} className="rounded-[16px] border border-white/10 bg-white/5 p-4">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div>
                        <p className="text-sm font-semibold text-white">{pickLocaleText(locale, item.titleID, item.titleEN)}</p>
                        <p className="text-xs text-white/60">{pickLocaleText(locale, item.subtitleID, item.subtitleEN)}</p>
                      </div>
                      <p className="text-lg font-bold text-purple-100">{pickLocaleText(locale, item.priceID, item.priceEN)}</p>
                    </div>

                    <Button
                      className="mt-3 w-full"
                      variant="secondary"
                      onClick={() => {
                        void handlePlanCheckout(item.planID);
                      }}
                      disabled={loadingKey !== null}
                    >
                      {loading ? pickLocaleText(locale, "Mengalihkan ke checkout...", "Redirecting to checkout...") : pickLocaleText(locale, "Upgrade paket", "Upgrade plan")}
                      {!loading && <ArrowRight className="ml-2 h-4 w-4" />}
                    </Button>
                  </div>
                );
              })}
            </div>
          </Card>
        </section>

        <section className="grid gap-4 xl:grid-cols-2">
          <Card>
            <div className="mb-3 flex items-center gap-2">
              <Clock3 className="h-4 w-4 text-amber-300" />
              <h3 className="text-sm font-semibold text-white">{pickLocaleText(locale, "Catatan Top-Up", "Top-Up Notes")}</h3>
            </div>
            <ul className="space-y-2 text-sm text-white/80">
              <li>• {pickLocaleText(locale, "Top-up berlaku lintas periode sampai menit habis.", "Top-up minutes carry over across periods until exhausted.")}</li>
              <li>• {pickLocaleText(locale, "Pemakaian voice otomatis memakai kuota paket dulu lalu saldo top-up.", "Voice usage consumes plan quota first, then top-up balance automatically.")}</li>
              <li>• {pickLocaleText(locale, "Jika pembayaran sukses, saldo top-up ter-update otomatis via webhook.", "If payment succeeds, top-up balance updates automatically via webhook.")}</li>
            </ul>
          </Card>

          <Card>
            <div className="mb-3 flex items-center gap-2">
              <ReceiptText className="h-4 w-4 text-emerald-300" />
              <h3 className="text-sm font-semibold text-white">{pickLocaleText(locale, "Status Penggunaan", "Usage Status")}</h3>
            </div>
            <div className="space-y-2 text-sm text-white/80">
              <p>{pickLocaleText(locale, `JD parse tersisa: ${status?.remaining_jd_parses ?? 0}`, `Remaining JD parses: ${status?.remaining_jd_parses ?? 0}`)}</p>
              <p>{pickLocaleText(locale, `Sesi tersisa periode ini: ${remainingSessionsLabel}`, `Remaining sessions this period: ${remainingSessionsLabel}`)}</p>
              <p>{pickLocaleText(locale, `Voice top-up terpakai: ${status?.used_voice_topup_minutes ?? 0} menit`, `Used top-up minutes: ${status?.used_voice_topup_minutes ?? 0} min`)}</p>
            </div>
          </Card>
        </section>

        {checkoutError && (
          <Card className="border border-red-400/30 bg-red-500/10">
            <p className="text-sm text-red-200">{checkoutError}</p>
          </Card>
        )}
      </div>
    </AppShell>
  );
}
