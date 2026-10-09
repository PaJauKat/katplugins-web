"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import type { User } from "@/lib/types";
import LoginButton from "./LoginButton";
import Logo from "./Logo";
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
    <header className="sticky top-0 z-40 border-b border-white/[0.08] bg-ink-950/80 backdrop-blur-xl">
      <div className="container-page flex h-16 items-center justify-between">
        <Logo />

        <nav className="hidden items-center gap-1 text-sm text-slate-400 md:flex">
          <Link
            href="/#plugins"
            className="rounded-lg px-3 py-2 transition hover:bg-white/[0.04] hover:text-white"
          >
            Plugins
          </Link>
          <Link
            href="/#precios"
            className="rounded-lg px-3 py-2 transition hover:bg-white/[0.04] hover:text-white"
          >
            Precios
          </Link>
          {user && (
            <Link
              href="/dashboard"
              className="rounded-lg px-3 py-2 transition hover:bg-white/[0.04] hover:text-white"
            >
              Mi panel
            </Link>
          )}
          {user && (
            <Link
              href="/admin"
              className="rounded-lg px-3 py-2 transition hover:bg-white/[0.04] hover:text-white"
            >
              Admin
            </Link>
          )}
        </nav>

        <div className="flex items-center gap-3">
          {!ready ? (
            <span className="h-9 w-28 animate-pulse rounded-lg bg-white/5" />
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
