"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import LoginButton from "./LoginButton";

const messages: Record<string, string> = {
  google: "Cancelaste o falló el inicio de sesión con Google.",
  state: "La sesión de login expiró. Intenta nuevamente.",
  token: "No se pudo validar la respuesta de Google.",
  email: "Tu cuenta de Google no entregó un correo.",
  db: "No se pudo crear tu perfil. Intenta más tarde.",
  session: "No se pudo crear la sesión. Intenta nuevamente.",
};

export default function LoginCard({
  next = "/dashboard",
  error,
}: {
  next?: string;
  error?: string;
}) {
  const router = useRouter();
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    api
      .me()
      .then(() => router.replace(next))
      .catch(() => setChecking(false));
  }, [next, router]);

  return (
    <div className="flex min-h-screen items-center justify-center px-5">
      <div className="card w-full max-w-md p-8">
        <Link href="/" className="flex items-center gap-2">
          <span className="grid h-9 w-9 place-items-center rounded-lg bg-brand font-black text-ink-950">
            K
          </span>
          <span className="text-lg font-bold text-white">
            Kat<span className="text-brand">Plugins</span>
          </span>
        </Link>

        <h1 className="mt-8 text-2xl font-bold text-white">Inicia sesión</h1>
        <p className="mt-2 text-sm text-slate-400">
          Usa tu cuenta de Google para acceder a tu panel y a tus plugins.
        </p>

        {error && messages[error] && (
          <p className="mt-4 rounded-lg border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-sm text-rose-300">
            {messages[error]}
          </p>
        )}

        <div className="mt-6">
          {checking ? (
            <div className="btn-primary w-full animate-pulse opacity-60">
              Cargando…
            </div>
          ) : (
            <LoginButton className="btn-primary w-full" next={next} />
          )}
        </div>

        <p className="mt-6 text-center text-xs text-slate-500">
          Al continuar aceptas el uso de tu cuenta para gestionar el acceso a los
          plugins.
        </p>
      </div>
    </div>
  );
}
