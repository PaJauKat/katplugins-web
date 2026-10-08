import LoginButton from "./LoginButton";

const plans = [
  {
    tier: "Free",
    price: "$0",
    period: "para siempre",
    tagline: "Para empezar",
    highlight: false,
    features: [
      "Plugins del plan Free",
      "Actualizaciones automaticas",
      "Soporte por Discord",
    ],
  },
  {
    tier: "Premium",
    price: "$5",
    period: "/ mes",
    tagline: "Lo mas elegido",
    highlight: true,
    features: [
      "Todos los plugins Free",
      "Plugins Premium (skilling y PvM)",
      "Actualizaciones automaticas",
      "Soporte prioritario",
    ],
  },
  {
    tier: "Pro",
    price: "$10",
    period: "/ mes",
    tagline: "Todo incluido",
    highlight: false,
    features: [
      "Todos los plugins Premium",
      "Plugins Pro (bosses avanzados)",
      "Acceso anticipado a novedades",
      "Soporte prioritario",
    ],
  },
];

export default function Pricing() {
  return (
    <section id="precios" className="container-page scroll-mt-20 py-20">
      <div className="mx-auto max-w-2xl text-center">
        <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
          Planes simples
        </h2>
        <p className="mt-3 text-slate-400">
          Elige tu plan y desbloquea los plugins. Los accesos se otorgan a tu
          cuenta de Google.
        </p>
      </div>

      <div className="mt-12 grid gap-6 md:grid-cols-3">
        {plans.map((plan) => (
          <div
            key={plan.tier}
            className={`card relative flex flex-col p-6 ${
              plan.highlight ? "border-brand/50 shadow-glow" : ""
            }`}
          >
            {plan.highlight && (
              <span className="absolute -top-3 left-6 rounded-full bg-brand px-3 py-1 text-xs font-bold text-ink-950">
                {plan.tagline}
              </span>
            )}
            <h3 className="text-lg font-semibold text-white">{plan.tier}</h3>
            <div className="mt-4 flex items-end gap-1">
              <span className="text-4xl font-black text-white">{plan.price}</span>
              <span className="pb-1 text-sm text-slate-400">{plan.period}</span>
            </div>
            {!plan.highlight && (
              <p className="mt-1 text-xs uppercase tracking-wide text-slate-500">
                {plan.tagline}
              </p>
            )}
            <ul className="mt-6 flex-1 space-y-3 text-sm text-slate-300">
              {plan.features.map((f) => (
                <li key={f} className="flex items-start gap-2">
                  <span className="mt-0.5 text-brand">✓</span>
                  {f}
                </li>
              ))}
            </ul>
            <div className="mt-6">
              {plan.tier === "Free" ? (
                <LoginButton
                  className="btn-primary w-full"
                  label="Empezar gratis"
                />
              ) : (
                <LoginButton
                  className={
                    plan.highlight
                      ? "btn-primary w-full"
                      : "btn-ghost w-full"
                  }
                  label="Iniciar sesión"
                />
              )}
            </div>
          </div>
        ))}
      </div>

      <p className="mt-6 text-center text-xs text-slate-500">
        Los pagos se gestionan manualmente. Al iniciar sesión, contacta al admin
        para activar tu plan.
      </p>
    </section>
  );
}
