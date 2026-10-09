import Image from "next/image";
import SiteNav from "@/components/SiteNav";
import PluginsGrid from "@/components/PluginsGrid";
import Pricing from "@/components/Pricing";
import LoginButton from "@/components/LoginButton";
import Logo from "@/components/Logo";

const highlights = [
  { title: "Skilling y PvM", text: "Plugins pensados para jugar mejor." },
  { title: "Acceso con Google", text: "Sin registros ni contraseñas." },
  { title: "Siempre al día", text: "Actualizaciones automáticas." },
];

export default function HomePage() {
  return (
    <div className="min-h-screen">
      <SiteNav />

      <main>
        {/* Hero */}
        <section className="container-page relative py-20 sm:py-28">
          <div className="mx-auto flex max-w-3xl flex-col items-center text-center">
            <span className="flex h-16 w-16 items-center justify-center rounded-2xl border border-white/10 bg-white/[0.03] shadow-glow">
              <Image
                src="/logo-k.png"
                alt="KatPlugins"
                width={44}
                height={36}
                priority
                className="h-9 w-auto"
              />
            </span>

            <span className="badge mt-6 border-brand/40 bg-brand/10 text-brand-light">
              Plugins para RuneLite
            </span>

            <h1 className="mt-5 text-4xl font-black leading-[1.05] tracking-tight text-white sm:text-6xl">
              Lleva tu RuneLite al
              <span className="text-brand"> siguiente nivel</span>
            </h1>
            <p className="mx-auto mt-5 max-w-xl text-base leading-relaxed text-slate-400 sm:text-lg">
              Skilling, bosses y calidad de vida. Activa tus plugins con tu
              cuenta de Google y mantenlos siempre actualizados.
            </p>
            <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
              <LoginButton />
              <a href="#plugins" className="btn-ghost">
                Ver plugins
              </a>
            </div>
          </div>

          <div className="mx-auto mt-16 grid max-w-4xl gap-px overflow-hidden rounded-xl border border-white/[0.08] bg-white/[0.06] sm:grid-cols-3">
            {highlights.map((h) => (
              <div key={h.title} className="bg-ink-900/90 px-6 py-5">
                <p className="text-sm font-semibold text-white">{h.title}</p>
                <p className="mt-1 text-sm text-slate-400">{h.text}</p>
              </div>
            ))}
          </div>
        </section>

        {/* Plugins */}
        <section id="plugins" className="container-page scroll-mt-20 py-16">
          <div className="flex flex-col gap-1">
            <h2 className="text-2xl font-bold tracking-tight text-white sm:text-3xl">
              Catálogo de plugins
            </h2>
            <p className="text-sm text-slate-400">
              Skilling, bosses y utilidades para tu cliente.
            </p>
          </div>

          <PluginsGrid />
        </section>

        <Pricing />
      </main>

      <footer className="border-t border-white/[0.08] py-10">
        <div className="container-page flex flex-col items-center justify-between gap-4 text-sm text-slate-500 sm:flex-row">
          <div className="flex flex-col items-center gap-3 sm:flex-row">
            <Logo />
            <span className="hidden text-slate-700 sm:inline">·</span>
            <p>© {new Date().getFullYear()} KatPlugins. No afiliado a Jagex.</p>
          </div>
          <div className="flex gap-5">
            <a
              href="https://ko-fi.com/pajau"
              target="_blank"
              rel="noreferrer"
              className="transition hover:text-white"
            >
              Donar
            </a>
            <a href="/#precios" className="transition hover:text-white">
              Precios
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}
