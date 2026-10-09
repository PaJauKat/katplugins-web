"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { BillingConfig, BillingInterval, BillingProviderId } from "@/lib/types";

function formatCLP(n: number) {
  return "$" + n.toLocaleString("es-CL");
}

const providerIcon: Record<BillingProviderId, string> = {
  flow: "Flow",
  lemonsqueezy: "Tarjeta",
  nowpayments: "Cripto",
};

export default function PlanCard() {
  const router = useRouter();
  const [config, setConfig] = useState<BillingConfig | null>(null);
  const [interval, setInterval] = useState<BillingInterval>("monthly");
  const [busy, setBusy] = useState<BillingProviderId | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.billing
      .config()
      .then(setConfig)
      .catch(() => setConfig(null));
  }, []);

  const prices = config?.plan.prices[interval];
  const providers = config?.providers ?? [];

  async function pay(provider: BillingProviderId) {
    setBusy(provider);
    setError(null);
    try {
      const res = await api.billing.checkout(provider, interval);
      window.location.href = res.url;
    } catch (e) {
      if ((e as { status?: number }).status === 401) {
        router.push("/login?next=/dashboard");
        return;
      }
      setError(e instanceof Error ? e.message : "No se pudo iniciar el pago.");
      setBusy(null);
    }
  }

  return (
    <div className="card relative flex flex-col border-brand/50 p-6 shadow-glow">
      <span className="absolute -top-3 left-6 rounded-full bg-brand px-3 py-1 text-xs font-bold text-white">
        Lo más elegido
      </span>
      <h3 className="text-lg font-semibold text-white">Plus</h3>

      <div className="mt-4 flex items-end gap-1">
        <span className="text-4xl font-black text-white">
          {prices ? formatCLP(prices.clp) : "$2.000"}
        </span>
        <span className="pb-1 text-sm text-slate-400">
          {interval === "annual" ? "/ año" : "/ mes"}
        </span>
      </div>
      <p className="mt-1 text-xs uppercase tracking-wide text-slate-500">
        {interval === "annual"
          ? "o US$30 al año con cripto"
          : "pago con Flow o tarjeta"}
      </p>

      <div className="mt-5 inline-flex rounded-lg border border-white/10 p-1 text-xs">
        {(["monthly", "annual"] as BillingInterval[]).map((i) => (
          <button
            key={i}
            type="button"
            onClick={() => setInterval(i)}
            className={`rounded-md px-3 py-1.5 font-semibold transition ${
              interval === i
                ? "bg-brand text-white"
                : "text-slate-400 hover:text-white"
            }`}
          >
            {i === "monthly" ? "Mensual" : "Anual"}
          </button>
        ))}
      </div>

      <ul className="mt-6 flex-1 space-y-3 text-sm text-slate-300">
        {[
          "Todos los plugins Free",
          "Plugins Plus (skilling y PvM)",
          "Actualizaciones automáticas",
          "Soporte prioritario",
        ].map((f) => (
          <li key={f} className="flex items-start gap-2">
            <span className="mt-0.5 text-brand">✓</span>
            {f}
          </li>
        ))}
      </ul>

      <div className="mt-6 space-y-2">
        {providers.map((p) => {
          const allowed = p.intervals.includes(interval);
          return (
            <button
              key={p.id}
              type="button"
              disabled={!p.enabled || !allowed || busy !== null}
              onClick={() => pay(p.id)}
              className="btn-primary w-full disabled:cursor-not-allowed disabled:opacity-40"
              title={!allowed ? "Solo disponible en el plan anual" : undefined}
            >
              {busy === p.id
                ? "Redirigiendo…"
                : `Pagar con ${providerIcon[p.id]}${prices ? ` · ${p.currency === "CLP" ? formatCLP(prices.clp) : `US$${prices.usd}`}` : ""}`}
            </button>
          );
        })}
        {providers.length === 0 && (
          <p className="text-center text-xs text-slate-500">
            Cargando métodos de pago…
          </p>
        )}
        {error && <p className="text-center text-xs text-rose-300">{error}</p>}
        <p className="text-center text-[11px] text-slate-500">
          Cripto solo disponible en el plan anual.
        </p>
      </div>
    </div>
  );
}
