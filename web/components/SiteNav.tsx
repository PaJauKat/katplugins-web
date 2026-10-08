"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import type { User } from "@/lib/types";
import LoginButton from "./LoginButton";
import SignOutButton from "./SignOutButton";

export default function SiteNav() {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    api
      .me()
      .then((res) => setUser(res.user))
      .catch(() => setUser(null))
      .finally(() => setReady(true));
  }, []);

  return (
    <header className="sticky top-0 z-40 border-b border-white/10 bg-ink-950/80 backdrop-blur">
      <div className="container-page flex h-16 items-center justify-between">
        <Link href="/" className="flex items-center gap-2">
          <span className="grid h-8 w-8 place-items-center rounded-lg bg-brand font-black text-ink-950">
            K
          </span>
          <span className="text-lg font-bold tracking-tight text-white">
            Kat<span className="text-brand">Plugins</span>
          </span>
        </Link>

        <nav className="hidden items-center gap-6 text-sm text-slate-300 md:flex">
          <Link href="/#plugins" className="transition hover:text-white">
            Plugins
          </Link>
          <Link href="/#precios" className="transition hover:text-white">
            Precios
          </Link>
          {user && (
            <Link href="/dashboard" className="transition hover:text-white">
              Mi panel
            </Link>
          )}
          {user && (
            <Link href="/admin" className="transition hover:text-white">
              Admin
            </Link>
          )}
        </nav>

        <div className="flex items-center gap-3">
          {!ready ? (
            <span className="h-8 w-24 animate-pulse rounded-lg bg-white/5" />
          ) : user ? (
            <>
              <span className="hidden text-sm text-slate-400 sm:inline">
                {user.email}
              </span>
              <SignOutButton />
            </>
          ) : (
            <LoginButton className="btn-primary btn-sm" label="Iniciar sesión" />
          )}
        </div>
      </div>
    </header>
  );
}
