"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { MeResponse, Tier } from "@/lib/types";
import PluginCard from "./PluginCard";

const tierClass: Record<Tier, string> = {
  free: "badge-free",
  premium: "badge-premium",
  pro: "badge-pro",
};

const tierName: Record<Tier, string> = {
  free: "Free",
  premium: "Premium",
  pro: "Pro",
};

export default function DashboardClient() {
  const router = useRouter();
  const [data, setData] = useState<MeResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api
      .me()
      .then(setData)
      .catch((e) => {
        if ((e as { status?: number }).status === 401) {
          router.replace("/login?next=/dashboard");
          return;
        }
        setError(e.message);
      })
      .finally(() => setLoading(false));
  }, [router]);

  if (loading) {
    return <p className="py-20 text-center text-slate-400">Cargando panel…</p>;
  }

  if (error || !data) {
    return (
      <div className="card p-8 text-center">
        <p className="text-rose-300">No se pudo cargar tu panel.</p>
        <p className="mt-2 text-sm text-slate-500">{error}</p>
      </div>
    );
  }

  const enabled = data.plugins.filter((p) => p.access).length;

  return (
    <div className="space-y-10">
      <section className="card flex flex-col items-start gap-5 p-6 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-4">
          {data.user.avatarUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={data.user.avatarUrl}
              alt=""
              className="h-14 w-14 rounded-full border border-white/10"
            />
          ) : (
            <div className="grid h-14 w-14 place-items-center rounded-full bg-brand text-xl font-black text-ink-950">
              {(data.user.fullName || data.user.email || "?").charAt(0)}
            </div>
          )}
          <div>
            <h1 className="text-xl font-bold text-white">
              {data.user.fullName || "Bienvenido"}
            </h1>
            <p className="text-sm text-slate-400">{data.user.email}</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <span className={tierClass[data.user.tier]}>
            Plan {tierName[data.user.tier]}
          </span>
          {data.isAdmin && (
            <Link href="/admin" className="btn-ghost btn-sm">
              Panel admin
            </Link>
          )}
        </div>
      </section>

      <section>
        <div className="mb-5 flex items-end justify-between">
          <div>
            <h2 className="text-lg font-bold text-white">Tus plugins</h2>
            <p className="text-sm text-slate-400">
              {enabled} de {data.plugins.length} habilitados.
            </p>
          </div>
        </div>

        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {data.plugins.map((pa) => (
            <PluginCard key={pa.plugin.id} plugin={pa.plugin} access={pa} />
          ))}
        </div>
      </section>

      <section className="card p-6 text-sm text-slate-400">
        <p>
          ¿Quieres cambiar de plan o desbloquear un plugin? Escríbenos y el
          administrador lo activará en tu cuenta.
        </p>
      </section>
    </div>
  );
}
