import LoginButton from "./LoginButton";
import PlanCard from "./PlanCard";

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

      <div className="mx-auto mt-12 grid max-w-3xl gap-6 md:grid-cols-2">
        <div className="card relative flex flex-col p-6">
          <h3 className="text-lg font-semibold text-white">Free</h3>
          <div className="mt-4 flex items-end gap-1">
            <span className="text-4xl font-black text-white">$0</span>
            <span className="pb-1 text-sm text-slate-400">para siempre</span>
          </div>
          <p className="mt-1 text-xs uppercase tracking-wide text-slate-500">
            Para empezar
          </p>
          <ul className="mt-6 flex-1 space-y-3 text-sm text-slate-300">
            {[
              "Plugins del plan Free",
              "Actualizaciones automáticas",
              "Soporte por Discord",
            ].map((f) => (
              <li key={f} className="flex items-start gap-2">
                <span className="mt-0.5 text-brand">✓</span>
                {f}
              </li>
            ))}
          </ul>
          <div className="mt-6">
            <LoginButton className="btn-ghost w-full" label="Empezar gratis" />
          </div>
        </div>

        <PlanCard />
      </div>

      <p className="mt-6 text-center text-xs text-slate-500">
        Los pagos se procesan con Flow (CLP), LemonSqueezy (USD) y NowPayments
        (cripto). Puedes cancelar cuando quieras.
      </p>
    </section>
  );
}
