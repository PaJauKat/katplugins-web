export type Tier = "free" | "premium" | "pro";
export type AccessSource = "tier" | "granted" | "revoked";

export interface User {
  id: string;
  email: string;
  fullName: string;
  avatarUrl: string;
  role: "user" | "admin";
  tier: Tier;
  createdAt?: string;
}

export interface Plugin {
  id: string;
  slug: string;
  name: string;
  description: string;
  requiredTier: Tier;
  active: boolean;
  sortOrder: number;
}

export interface PluginAccess {
  plugin: Plugin;
  access: boolean;
  source: AccessSource;
  override?: boolean;
}

export interface MeResponse {
  user: User;
  isAdmin: boolean;
  plugins: PluginAccess[];
}

export interface AdminUser {
  user: User;
  grantedCount: number;
  revokedCount: number;
}

export const TIERS: { id: Tier; name: string; price: string; tagline: string }[] = [
  { id: "free", name: "Free", price: "$0", tagline: "Para empezar" },
  { id: "premium", name: "Premium", price: "$5", tagline: "Lo mas elegido" },
  { id: "pro", name: "Pro", price: "$10", tagline: "Todo incluido" },
];
