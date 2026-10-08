import type { Plugin, PluginAccess, Tier } from "@/lib/types";

const tierBadge: Record<Tier, string> = {
  free: "badge-free",
  premium: "badge-premium",
  pro: "badge-pro",
};

const tierLabel: Record<Tier, string> = {
  free: "Free",
  premium: "Premium",
  pro: "Pro",
};

export default function PluginCard({
  plugin,
  access,
}: {
  plugin: Plugin;
  access?: PluginAccess;
}) {
  const locked = access ? !access.access : false;

  return (
    <article className="card flex h-full flex-col gap-3 p-5 transition hover:border-brand/40">
      <div className="flex items-start justify-between gap-3">
        <h3 className="text-base font-semibold text-white">{plugin.name}</h3>
        <span className={tierBadge[plugin.requiredTier]}>
          {tierLabel[plugin.requiredTier]}
        </span>
      </div>
      <p className="flex-1 text-sm leading-relaxed text-slate-400">
        {plugin.description || "Plugin de RuneLite."}
      </p>
      <div className="flex items-center justify-between border-t border-white/5 pt-3 text-xs">
        <span className="font-mono text-slate-500">{plugin.slug}</span>
        {access ? (
          access.access ? (
            <span className="font-semibold text-emerald-400">Habilitado</span>
          ) : (
            <span className="font-semibold text-rose-400">Bloqueado</span>
          )
        ) : locked ? (
          <span className="font-semibold text-rose-400">Bloqueado</span>
        ) : null}
      </div>
    </article>
  );
}
