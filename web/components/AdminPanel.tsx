"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { AdminUser, MeResponse, Plugin, Tier } from "@/lib/types";

const TIERS: Tier[] = ["free", "premium", "pro"];

const tierClass: Record<Tier, string> = {
  free: "badge-free",
  premium: "badge-premium",
  pro: "badge-pro",
};

type Status = { type: "ok" | "error"; msg: string } | null;

const emptyPlugin: Plugin = {
  id: "",
  slug: "",
  name: "",
  description: "",
  requiredTier: "free",
  active: true,
  sortOrder: 0,
};

export default function AdminPanel() {
  const router = useRouter();
  const [tab, setTab] = useState<"users" | "plugins">("users");
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [selected, setSelected] = useState<AdminUser | null>(null);
  const [access, setAccess] = useState<MeResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [denied, setDenied] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);
  const [status, setStatus] = useState<Status>(null);
  const [draft, setDraft] = useState<Plugin>(emptyPlugin);

  const loadUsers = useCallback(async () => {
    const res = await api.admin.users();
    setUsers(res.users);
  }, []);

  const loadPlugins = useCallback(async () => {
    const res = await api.admin.plugins();
    setPlugins(res.plugins);
  }, []);

  useEffect(() => {
    (async () => {
      try {
        await Promise.all([loadUsers(), loadPlugins()]);
      } catch (e) {
        if ((e as { status?: number }).status === 401) {
          router.replace("/login?next=/admin");
          return;
        }
        setDenied(true);
        setStatus({
          type: "error",
          msg: e instanceof Error ? e.message : "Error de acceso",
        });
      } finally {
        setLoading(false);
      }
    })();
  }, [loadUsers, loadPlugins, router]);

  function flash(type: "ok" | "error", msg: string) {
    setStatus({ type, msg });
    setTimeout(() => setStatus(null), 3500);
  }

  async function changeTier(userId: string, tier: Tier) {
    setBusy(userId);
    try {
      await api.admin.setTier(userId, tier);
      setUsers((prev) =>
        prev.map((u) =>
          u.user.id === userId ? { ...u, user: { ...u.user, tier } } : u,
        ),
      );
      if (selected?.user.id === userId) {
        setSelected((s) =>
          s ? { ...s, user: { ...s.user, tier } } : s,
        );
      }
      flash("ok", "Tier actualizado.");
    } catch (e) {
      flash("error", e instanceof Error ? e.message : "Error");
    } finally {
      setBusy(null);
    }
  }

  async function openUser(u: AdminUser) {
    setSelected(u);
    setAccess(null);
    try {
      const res = await api.admin.userAccess(u.user.id);
      setAccess(res);
    } catch (e) {
      flash("error", e instanceof Error ? e.message : "Error");
    }
  }

  async function handleSetAccess(
    pluginId: string,
    granted: boolean | null,
    key: string,
  ) {
    if (!selected) return;
    setBusy(key);
    try {
      await api.admin.setAccess(selected.user.id, pluginId, granted);
      const res = await api.admin.userAccess(selected.user.id);
      setAccess(res);
      await loadUsers();
    } catch (e) {
      flash("error", e instanceof Error ? e.message : "Error");
    } finally {
      setBusy(null);
    }
  }

  async function savePlugin(plugin: Plugin) {
    setBusy(`plugin-${plugin.id || plugin.slug}`);
    try {
      await api.admin.upsertPlugin(plugin);
      await loadPlugins();
      flash("ok", "Plugin guardado.");
    } catch (e) {
      flash("error", e instanceof Error ? e.message : "Error");
    } finally {
      setBusy(null);
    }
  }

  async function deletePlugin(plugin: Plugin) {
    if (!confirm(`¿Eliminar el plugin "${plugin.name}"?`)) return;
    setBusy(`plugin-${plugin.id}`);
    try {
      await api.admin.deletePlugin(plugin.id);
      await loadPlugins();
      flash("ok", "Plugin eliminado.");
    } catch (e) {
      flash("error", e instanceof Error ? e.message : "Error");
    } finally {
      setBusy(null);
    }
  }

  if (loading) {
    return <p className="py-20 text-center text-slate-400">Cargando…</p>;
  }

  if (denied) {
    return (
      <div className="card p-8 text-center">
        <h1 className="text-xl font-bold text-white">Acceso restringido</h1>
        <p className="mt-2 text-sm text-slate-400">
          Esta sección es solo para administradores.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-white">Panel de administración</h1>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => setTab("users")}
            className={tab === "users" ? "btn-primary btn-sm" : "btn-ghost btn-sm"}
          >
            Usuarios
          </button>
          <button
            type="button"
            onClick={() => setTab("plugins")}
            className={
              tab === "plugins" ? "btn-primary btn-sm" : "btn-ghost btn-sm"
            }
          >
            Plugins
          </button>
        </div>
      </div>

      {status && (
        <div
          className={`rounded-lg border px-4 py-2 text-sm ${
            status.type === "ok"
              ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-300"
              : "border-rose-500/30 bg-rose-500/10 text-rose-300"
          }`}
        >
          {status.msg}
        </div>
      )}

      {tab === "users" ? (
        <UsersTab
          users={users}
          selected={selected}
          access={access}
          busy={busy}
          onTier={changeTier}
          onOpen={openUser}
          onClose={() => {
            setSelected(null);
            setAccess(null);
          }}
          onSetAccess={handleSetAccess}
        />
      ) : (
        <PluginsTab
          plugins={plugins}
          busy={busy}
          draft={draft}
          setDraft={setDraft}
          onSave={savePlugin}
          onDelete={deletePlugin}
        />
      )}
    </div>
  );
}

function UsersTab({
  users,
  selected,
  access,
  busy,
  onTier,
  onOpen,
  onClose,
  onSetAccess,
}: {
  users: AdminUser[];
  selected: AdminUser | null;
  access: MeResponse | null;
  busy: string | null;
  onTier: (userId: string, tier: Tier) => void;
  onOpen: (u: AdminUser) => void;
  onClose: () => void;
  onSetAccess: (
    pluginId: string,
    granted: boolean | null,
    key: string,
  ) => void;
}) {
  return (
    <div className="grid gap-6 lg:grid-cols-[1.4fr_1fr]">
      <div className="card overflow-hidden">
        <table className="w-full text-left text-sm">
          <thead className="bg-white/5 text-xs uppercase tracking-wide text-slate-400">
            <tr>
              <th className="px-4 py-3">Usuario</th>
              <th className="px-4 py-3">Tier</th>
              <th className="px-4 py-3 text-right">Accesos</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr
                key={u.user.id}
                className="border-t border-white/5 hover:bg-white/[0.03]"
              >
                <td className="px-4 py-3">
                  <div className="font-medium text-white">
                    {u.user.fullName || "(sin nombre)"}
                  </div>
                  <div className="text-xs text-slate-500">{u.user.email}</div>
                </td>
                <td className="px-4 py-3">
                  <select
                    value={u.user.tier}
                    disabled={busy === u.user.id}
                    onChange={(e) => onTier(u.user.id, e.target.value as Tier)}
                    className="input w-auto py-1 text-xs"
                  >
                    {TIERS.map((t) => (
                      <option key={t} value={t}>
                        {t}
                      </option>
                    ))}
                  </select>
                </td>
                <td className="px-4 py-3 text-right text-xs">
                  <span className="text-emerald-400">+{u.grantedCount}</span>
                  {" / "}
                  <span className="text-rose-400">-{u.revokedCount}</span>
                </td>
                <td className="px-4 py-3 text-right">
                  <button
                    type="button"
                    className="btn-ghost btn-sm"
                    onClick={() => onOpen(u)}
                  >
                    Gestionar
                  </button>
                </td>
              </tr>
            ))}
            {users.length === 0 && (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-slate-500">
                  Aun no hay usuarios registrados.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="card p-5">
        {!selected ? (
          <p className="py-12 text-center text-sm text-slate-500">
            Selecciona un usuario para gestionar sus plugins.
          </p>
        ) : (
          <div className="space-y-4">
            <div className="flex items-start justify-between">
              <div>
                <h2 className="font-semibold text-white">
                  {selected.user.fullName || selected.user.email}
                </h2>
                <p className="text-xs text-slate-500">{selected.user.email}</p>
                <span className={`${tierClass[selected.user.tier]} mt-2`}>
                  {selected.user.tier}
                </span>
              </div>
              <button
                type="button"
                className="text-slate-500 hover:text-white"
                onClick={onClose}
              >
                ✕
              </button>
            </div>

            {!access ? (
              <p className="py-8 text-center text-sm text-slate-500">
                Cargando accesos…
              </p>
            ) : (
              <ul className="max-h-[28rem] space-y-2 overflow-auto pr-1">
                {access.plugins.map((pa) => {
                  const key = `${selected.user.id}-${pa.plugin.id}`;
                  return (
                    <li
                      key={pa.plugin.id}
                      className="rounded-lg border border-white/5 bg-ink-850/60 p-3"
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-medium text-white">
                          {pa.plugin.name}
                        </span>
                        <span
                          className={
                            pa.access
                              ? "text-xs font-semibold text-emerald-400"
                              : "text-xs font-semibold text-rose-400"
                          }
                        >
                          {pa.access ? "Habilitado" : "Bloqueado"}
                        </span>
                      </div>
                      <div className="mt-1 flex items-center justify-between">
                        <span className="text-[11px] uppercase tracking-wide text-slate-500">
                          {pa.source === "tier"
                            ? `por tier (${pa.plugin.requiredTier})`
                            : pa.source === "granted"
                              ? "concedido manual"
                              : "revocado manual"}
                        </span>
                        <div className="flex gap-1">
                          <button
                            type="button"
                            disabled={busy === key}
                            onClick={() => onSetAccess(pa.plugin.id, null, key)}
                            className={`btn-sm ${
                              pa.source === "tier"
                                ? "btn-primary"
                                : "btn-ghost"
                            }`}
                          >
                            Auto
                          </button>
                          <button
                            type="button"
                            disabled={busy === key}
                            onClick={() => onSetAccess(pa.plugin.id, true, key)}
                            className={`btn-sm ${
                              pa.source === "granted"
                                ? "btn-primary"
                                : "btn-ghost"
                            }`}
                          >
                            Dar
                          </button>
                          <button
                            type="button"
                            disabled={busy === key}
                            onClick={() => onSetAccess(pa.plugin.id, false, key)}
                            className={`btn-sm ${
                              pa.source === "revoked"
                                ? "btn-primary"
                                : "btn-ghost"
                            }`}
                          >
                            Quitar
                          </button>
                        </div>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function PluginsTab({
  plugins,
  busy,
  draft,
  setDraft,
  onSave,
  onDelete,
}: {
  plugins: Plugin[];
  busy: string | null;
  draft: Plugin;
  setDraft: (p: Plugin) => void;
  onSave: (p: Plugin) => void;
  onDelete: (p: Plugin) => void;
}) {
  return (
    <div className="space-y-6">
      <div className="card p-5">
        <h2 className="mb-4 font-semibold text-white">
          {draft.id ? "Editar plugin" : "Nuevo plugin"}
        </h2>
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="label">Slug</label>
            <input
              className="input"
              value={draft.slug}
              onChange={(e) => setDraft({ ...draft, slug: e.target.value })}
              placeholder="spec-key"
            />
          </div>
          <div>
            <label className="label">Nombre</label>
            <input
              className="input"
              value={draft.name}
              onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              placeholder="Spec Key"
            />
          </div>
          <div className="sm:col-span-2">
            <label className="label">Descripción</label>
            <input
              className="input"
              value={draft.description}
              onChange={(e) =>
                setDraft({ ...draft, description: e.target.value })
              }
            />
          </div>
          <div>
            <label className="label">Tier requerido</label>
            <select
              className="input"
              value={draft.requiredTier}
              onChange={(e) =>
                setDraft({ ...draft, requiredTier: e.target.value as Tier })
              }
            >
              {TIERS.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
          </div>
          <div className="flex items-end gap-4">
            <label className="flex items-center gap-2 text-sm text-slate-300">
              <input
                type="checkbox"
                checked={draft.active}
                onChange={(e) =>
                  setDraft({ ...draft, active: e.target.checked })
                }
              />
              Activo
            </label>
            <div className="flex-1">
              <label className="label">Orden</label>
              <input
                type="number"
                className="input"
                value={draft.sortOrder}
                onChange={(e) =>
                  setDraft({ ...draft, sortOrder: Number(e.target.value) })
                }
              />
            </div>
          </div>
        </div>
        <div className="mt-4 flex gap-2">
          <button
            type="button"
            className="btn-primary"
            disabled={!draft.slug || !draft.name}
            onClick={() => onSave(draft)}
          >
            {draft.id ? "Guardar cambios" : "Crear plugin"}
          </button>
          {draft.id && (
            <button
              type="button"
              className="btn-ghost"
              onClick={() => setDraft({ ...emptyPlugin, id: draft.id })}
            >
              Cancelar
            </button>
          )}
        </div>
      </div>

      <div className="card overflow-hidden">
        <table className="w-full text-left text-sm">
          <thead className="bg-white/5 text-xs uppercase tracking-wide text-slate-400">
            <tr>
              <th className="px-4 py-3">Plugin</th>
              <th className="px-4 py-3">Tier</th>
              <th className="px-4 py-3">Estado</th>
              <th className="px-4 py-3" />
            </tr>
          </thead>
          <tbody>
            {plugins.map((p) => (
              <tr key={p.id} className="border-t border-white/5">
                <td className="px-4 py-3">
                  <div className="font-medium text-white">{p.name}</div>
                  <div className="font-mono text-xs text-slate-500">
                    {p.slug}
                  </div>
                </td>
                <td className="px-4 py-3">
                  <span className={tierClass[p.requiredTier]}>
                    {p.requiredTier}
                  </span>
                </td>
                <td className="px-4 py-3 text-xs">
                  {p.active ? (
                    <span className="text-emerald-400">Activo</span>
                  ) : (
                    <span className="text-slate-500">Inactivo</span>
                  )}
                </td>
                <td className="px-4 py-3 text-right">
                  <div className="flex justify-end gap-2">
                    <button
                      type="button"
                      className="btn-ghost btn-sm"
                      onClick={() => setDraft(p)}
                    >
                      Editar
                    </button>
                    <button
                      type="button"
                      className="btn-ghost btn-sm text-rose-300"
                      disabled={busy === `plugin-${p.id}`}
                      onClick={() => onDelete(p)}
                    >
                      Eliminar
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
