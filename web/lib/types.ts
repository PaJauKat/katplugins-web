export type Tier = "free" | "plus" | "pro";
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
  { id: "plus", name: "Plus", price: "$3", tagline: "Lo mas elegido" },
  { id: "pro", name: "Pro", price: "$10", tagline: "Todo incluido" },
];

export type BillingInterval = "monthly" | "annual";
export type BillingProviderId = "flow" | "lemonsqueezy" | "nowpayments";

export interface BillingProvider {
  id: BillingProviderId;
  name: string;
  currency: "CLP" | "USD";
  intervals: BillingInterval[];
  enabled: boolean;
}

export interface BillingConfig {
  plan: {
    id: "plus";
    name: string;
    prices: {
      monthly: { clp: number; usd: number };
      annual: { clp: number; usd: number };
    };
  };
  providers: BillingProvider[];
}

export interface Subscription {
  id: string;
  provider: string;
  plan: string;
  interval: string;
  status: string;
  amount: number;
  currency: string;
  current_period_end?: string;
  created_at?: string;
}

export interface BillingStatus {
  tier: Tier;
  subscriptions: Subscription[];
}
