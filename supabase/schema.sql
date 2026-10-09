-- ============================================================
-- KatPlugins — Esquema de base de datos (Supabase / Postgres)
-- Auth: Google OAuth directo (gestionado por el backend Go).
-- Supabase se usa SOLO como base de datos (PostgREST + service role).
--
-- Ejecutar en: Supabase Dashboard -> SQL Editor -> New query
-- Es idempotente y tambien migra un esquema previo basado en Supabase Auth.
-- ============================================================

create extension if not exists "pgcrypto";

-- ------------------------------------------------------------
-- Limpieza de un esquema anterior basado en Supabase Auth
-- (no borra datos; solo remueve acoplamientos).
-- El DROP FUNCTION ... CASCADE tambien elimina los triggers asociados.
-- ------------------------------------------------------------
drop trigger if exists on_auth_user_created on auth.users;
drop function if exists public.handle_new_user();
drop function if exists public.protect_profile_fields() cascade;
drop function if exists public.is_admin() cascade;

-- ------------------------------------------------------------
-- Tipos
-- ------------------------------------------------------------
do $$ begin
  create type public.user_role as enum ('user', 'admin');
exception when duplicate_object then null; end $$;

do $$ begin
  create type public.subscription_tier as enum ('free', 'plus', 'pro');
exception when duplicate_object then null; end $$;

-- Migracion: renombra el valor 'premium' a 'plus' en esquemas existentes.
do $$ begin
  if exists (
    select 1 from pg_enum e join pg_type t on t.oid = e.enumtypid
    where t.typname = 'subscription_tier' and e.enumlabel = 'premium'
  ) and not exists (
    select 1 from pg_enum e join pg_type t on t.oid = e.enumtypid
    where t.typname = 'subscription_tier' and e.enumlabel = 'plus'
  ) then
    alter type public.subscription_tier rename value 'premium' to 'plus';
  end if;
end $$;

-- ------------------------------------------------------------
-- Perfiles (identidad propia, ligada a Google por google_sub)
-- ------------------------------------------------------------
create table if not exists public.profiles (
  id          uuid primary key default gen_random_uuid(),
  email       text not null,
  google_sub  text,
  full_name   text,
  avatar_url  text,
  role        public.user_role       not null default 'user',
  tier        public.subscription_tier not null default 'free',
  created_at  timestamptz not null default now(),
  updated_at  timestamptz not null default now()
);

-- Migracion desde el esquema antiguo (basado en auth.users)
alter table public.profiles drop constraint if exists profiles_id_fkey;
alter table public.profiles alter column id drop default;
alter table public.profiles alter column id set default gen_random_uuid();
alter table public.profiles add column if not exists google_sub text;
alter table public.profiles add column if not exists flow_customer_id text;
create unique index if not exists profiles_email_key on public.profiles (lower(email));
create unique index if not exists profiles_google_sub_key on public.profiles (google_sub) where google_sub is not null;

-- ------------------------------------------------------------
-- Plugins (catalogo)
-- required_tier = tier minimo que da acceso automatico
-- ------------------------------------------------------------
create table if not exists public.plugins (
  id             uuid primary key default gen_random_uuid(),
  slug           text not null unique,
  name           text not null,
  description    text not null default '',
  required_tier  public.subscription_tier not null default 'free',
  active         boolean not null default true,
  sort_order     int  not null default 0,
  created_at     timestamptz not null default now()
);

-- Metadatos de empaquetado (los rellena el build de KatPlugins via /api/admin/plugins/sync)
alter table public.plugins add column if not exists package  text not null default '';
alter table public.plugins add column if not exists jar_file text not null default '';
alter table public.plugins add column if not exists version  text not null default '';
alter table public.plugins add column if not exists sha256   text not null default '';

-- ------------------------------------------------------------
-- Accesos explicitos por usuario/plugin (override del tier)
-- granted = true  -> habilitado aunque su tier no lo incluya
-- granted = false -> bloqueado aunque su tier si lo incluya
-- ------------------------------------------------------------
create table if not exists public.plugin_access (
  user_id    uuid not null references public.profiles(id) on delete cascade,
  plugin_id  uuid not null references public.plugins(id)  on delete cascade,
  granted    boolean not null default true,
  updated_at timestamptz not null default now(),
  primary key (user_id, plugin_id)
);

-- ------------------------------------------------------------
-- Auditoria de acciones de admin
-- ------------------------------------------------------------
create table if not exists public.audit_log (
  id             bigint generated by default as identity primary key,
  actor_id       uuid references public.profiles(id) on delete set null,
  action         text not null,
  target_user_id uuid,
  plugin_id      uuid,
  detail         jsonb not null default '{}'::jsonb,
  created_at     timestamptz not null default now()
);

-- ------------------------------------------------------------
-- Suscripciones (Flow / LemonSqueezy / NowPayments)
-- El backend activa el tier 'plus' mientras exista una suscripcion
-- activa y no vencida (current_period_end).
-- ------------------------------------------------------------
create table if not exists public.subscriptions (
  id                 uuid primary key default gen_random_uuid(),
  user_id            uuid not null references public.profiles(id) on delete cascade,
  provider           text not null,               -- flow | lemonsqueezy | nowpayments
  plan               text not null default 'plus',
  interval           text not null,               -- monthly | annual
  status             text not null default 'pending', -- pending|active|past_due|cancelled|expired
  external_id        text,
  checkout_ref       text,
  amount             numeric(12,2) not null default 0,
  currency           text not null default 'CLP',
  current_period_end timestamptz,
  created_at         timestamptz not null default now(),
  updated_at         timestamptz not null default now()
);

create index if not exists subscriptions_user_idx on public.subscriptions (user_id);
create unique index if not exists subscriptions_external_key on public.subscriptions (provider, external_id) where external_id is not null;

-- ------------------------------------------------------------
-- Helpers
-- ------------------------------------------------------------
create or replace function public.tier_rank(t public.subscription_tier)
returns int language sql immutable as $$
  select case t
    when 'free' then 0
    when 'plus' then 1
    when 'pro' then 2
    else 0
  end;
$$;

create or replace function public.set_updated_at()
returns trigger language plpgsql as $$
begin
  new.updated_at = now();
  return new;
end; $$;

drop trigger if exists trg_profiles_updated on public.profiles;
create trigger trg_profiles_updated before update on public.profiles
  for each row execute function public.set_updated_at();

drop trigger if exists trg_plugin_access_updated on public.plugin_access;
create trigger trg_plugin_access_updated before update on public.plugin_access
  for each row execute function public.set_updated_at();

drop trigger if exists trg_subscriptions_updated on public.subscriptions;
create trigger trg_subscriptions_updated before update on public.subscriptions
  for each row execute function public.set_updated_at();

-- ------------------------------------------------------------
-- Seguridad
-- El backend Go usa la secret key (service_role), que bypassa RLS.
-- No se expone la base a anon/authenticated: RLS habilitado sin
-- politicas = acceso denegado, y sin grants para esos roles.
-- ------------------------------------------------------------
alter table public.profiles      enable row level security;
alter table public.plugins       enable row level security;
alter table public.plugin_access enable row level security;
alter table public.audit_log     enable row level security;
alter table public.subscriptions enable row level security;

do $$ begin
  revoke all on public.profiles, public.plugins, public.plugin_access, public.audit_log, public.subscriptions from anon, authenticated;
end $$;

grant usage on schema public to service_role;
grant all on public.profiles, public.plugins, public.plugin_access, public.audit_log, public.subscriptions to service_role;
grant all on all sequences in schema public to service_role;

-- Elimina politicas heredadas del esquema anterior (ya no aplican)
drop policy if exists "profiles_select_own_or_admin" on public.profiles;
drop policy if exists "profiles_update_own" on public.profiles;
drop policy if exists "plugins_select_active" on public.plugins;
drop policy if exists "plugins_admin_all" on public.plugins;
drop policy if exists "plugin_access_select_own_or_admin" on public.plugin_access;
drop policy if exists "plugin_access_admin_all" on public.plugin_access;
drop policy if exists "audit_admin_select" on public.audit_log;

-- ------------------------------------------------------------
-- Seed del catalogo de plugins (los metadatos de empaquetado los
-- rellena el build con `gradlew DeployPlugins`).
-- NOTA: com/safe/guitarHero (Mania) va siempre en katcore.jar y no
-- se gatea (es clase base de muchos plugins). Cada plugin es su
-- propio jar; kattob requiere que katparty tambien este habilitado.
-- ------------------------------------------------------------
insert into public.plugins (slug, name, description, required_tier, sort_order, package) values
  ('blackjackleftclick', 'Blackjack LeftClick', '', 'free',     1,  'com.safe.blackjackleftclick'),
  ('clueanswers',        'Clue Answers',        '', 'free',     2,  'com.safe.clueanswers'),
  ('devtools',           'Developer Tools',     '', 'free',     3,  'com.safe.devtools'),
  ('neverlogout',        'NeverLogout',         '', 'free',     4,  'com.safe.neverlogout'),
  ('playeratktimer',     'Player Attack Timer', '', 'free',     5,  'com.safe.playerAtkTimer'),
  ('alchemicalhydra',    'Alchemical Hydra',    '', 'plus', 10,  'com.safe.alchemicalhydra'),
  ('brutus',             'Brutus',              '', 'plus', 11,  'com.safe.brutus'),
  ('clueswaps',          'Clue Swaps',          '', 'plus', 12,  'com.safe.clueswaps'),
  ('coxcmchest',         'Cox CM chest',        '', 'plus', 13,  'com.safe.coxcmchest'),
  ('coxcmchestsimple',   'Cox CM Chest simple', '', 'plus', 14,  'com.safe.coxcmchestsimple'),
  ('coxkat',             'Cox Kat',             '', 'plus', 15,  'com.safe.coxkat'),
  ('duke',               'Duke',                '', 'plus', 16,  'com.safe.duke'),
  ('emoteskat',          'Emotes',              '', 'plus', 17,  'com.safe.emoteskat'),
  ('faldita',            'Nightmare',           '', 'plus', 18,  'com.safe.faldita'),
  ('generalpvm',         'General pvm',         '', 'plus', 19,  'com.safe.generalPvm'),
  ('hueycoatl',          'Hueycoatl',           '', 'plus', 21,  'com.safe.hueycoatl'),
  ('hydra',              'Hydra Helper',        '', 'plus', 22,  'com.safe.hydra'),
  ('jads',               'Jads',                '', 'plus', 23,  'com.safe.jads'),
  ('levi',               'Levi',                '', 'plus', 24,  'com.safe.levi'),
  ('madangel',           'Mad Angel',           '', 'plus', 25,  'com.safe.madangel'),
  ('maggotkingkat',      'Maggot Kat',          '', 'plus', 26,  'com.safe.maggotkingkat'),
  ('mirror',             'Mirror',              '', 'plus', 27,  'com.prohibidos.mirror'),
  ('mokhaiotl',          'Mokhaiotl',           '', 'plus', 28,  'com.safe.mokhaiotl'),
  ('sepulchrekat',       'Sepulchre Kat',       '', 'plus', 29,  'com.safe.sepulchrekat'),
  ('tormenteddemons',    'Tormented demons',    '', 'plus', 30,  'com.safe.tormentedDemons'),
  ('vardorvis',          'Vardorvis',           '', 'plus', 31,  'com.safe.vardorvis'),
  ('yama',               'Yama',                '', 'plus', 32,  'com.safe.yama'),
  ('coliseo',            'Coliseo',             '', 'pro',     40,  'com.safe.coliseo'),
  ('infernal',           'Inferno',             '', 'pro',     41,  'com.safe.infernal'),
  ('katparty',           'PartyKat',            '', 'pro',     42,  'com.privado.katparty'),
  ('kattob',             'Kat ToB',             '', 'pro',     43,  'com.privado.kattob'),
  ('theatre',            'Teatro Sangriento',   '', 'pro',     44,  'com.safe.theatre'),
  ('toacito',            'Toa Kat',             '', 'pro',     45,  'com.safe.Toacito')
on conflict (slug) do nothing;

-- ------------------------------------------------------------
-- Bootstrap del admin
-- Tras tu primer login con Google, ejecuta (con tu correo):
--   update public.profiles set role = 'admin' where lower(email) = lower('tu-correo@gmail.com');
-- (o define ADMIN_EMAILS en las variables de entorno del backend)
-- ------------------------------------------------------------
