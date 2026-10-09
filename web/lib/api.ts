import type {
  AdminUser,
  BillingConfig,
  BillingInterval,
  BillingStatus,
  MeResponse,
  Plugin,
  PluginAccess,
} from "@/lib/types";

// En produccion (Vercel Services) la API vive en el mismo origen bajo /api.
// Si despliegas el backend aparte, define NEXT_PUBLIC_API_BASE con su URL.
const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "";

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body) headers.set("Content-Type", "application/json");

  const res = await fetch(`${API_BASE}/api${path}`, {
    ...init,
    headers,
    credentials: "include",
    cache: "no-store",
  });

  const body = (await res.json().catch(() => ({}))) as Record<string, unknown>;
  if (!res.ok) {
    const message =
      typeof body.error === "string" ? body.error : `Error ${res.status}`;
    const error = new Error(message) as Error & { status?: number };
    error.status = res.status;
    throw error;
  }
  return body as T;
}

export const api = {
  me: () => apiFetch<MeResponse>("/me"),
  plugins: () => apiFetch<{ plugins: PluginAccess[] }>("/plugins"),
  logout: () =>
    apiFetch<{ ok: boolean }>("/auth/logout", { method: "POST" }),
  billing: {
    config: () => apiFetch<BillingConfig>("/billing/providers"),
    status: () => apiFetch<BillingStatus>("/billing/subscription"),
    checkout: (provider: string, interval: BillingInterval) =>
      apiFetch<{ url: string }>("/billing/checkout", {
        method: "POST",
        body: JSON.stringify({ provider, interval }),
      }),
  },
  admin: {
    users: () => apiFetch<{ users: AdminUser[] }>("/admin/users"),
    setTier: (userId: string, tier: string) =>
      apiFetch<{ ok: boolean }>("/admin/users/tier", {
        method: "POST",
        body: JSON.stringify({ userId, tier }),
      }),
    userAccess: (userId: string) =>
      apiFetch<MeResponse>(
        `/admin/users/access?userId=${encodeURIComponent(userId)}`,
      ),
    setAccess: (userId: string, pluginId: string, granted: boolean | null) =>
      apiFetch<{ ok: boolean }>("/admin/access", {
        method: "POST",
        body: JSON.stringify({ userId, pluginId, granted }),
      }),
    plugins: () => apiFetch<{ plugins: Plugin[] }>("/admin/plugins"),
    upsertPlugin: (plugin: Partial<Plugin>) =>
      apiFetch<{ ok: boolean }>("/admin/plugins", {
        method: "POST",
        body: JSON.stringify({
          id: plugin.id ?? "",
          slug: plugin.slug ?? "",
          name: plugin.name ?? "",
          description: plugin.description ?? "",
          requiredTier: plugin.requiredTier ?? "free",
          active: plugin.active,
          sortOrder: plugin.sortOrder,
        }),
      }),
    deletePlugin: (id: string) =>
      apiFetch<{ ok: boolean }>(`/admin/plugins?id=${encodeURIComponent(id)}`, {
        method: "DELETE",
      }),
  },
};
