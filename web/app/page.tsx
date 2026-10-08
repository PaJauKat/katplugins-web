import SiteNav from "@/components/SiteNav";
import PluginsGrid from "@/components/PluginsGrid";
import Pricing from "@/components/Pricing";
import LoginButton from "@/components/LoginButton";

export default function HomePage() {
  return (
    <div className="min-h-screen">
      <SiteNav />

      <main>
        {/* Hero */}
        <section className="container-page relative py-20 sm:py-28">
          <div className="mx-auto max-w-3xl text-center">
            <span className="badge border-brand/40 bg-brand/10 text-brand">
              Plugins para RuneLite
            </span>
            <h1 className="mt-5 text-4xl font-black leading-tight tracking-tight text-white sm:text-6xl">
              Lleva tu RuneLite al
              <span className="text-brand"> siguiente nivel</span>
            </h1>
            <p className="mx-auto mt-5 max-w-xl text-lg text-slate-400">
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
        </section>

        {/* Plugins */}
        <section id="plugins" className="container-page scroll-mt-20 py-12">
          <div>
            <h2 className="text-2xl font-bold text-white sm:text-3xl">
              Catalogo de plugins
            </h2>
            <p className="mt-2 text-sm text-slate-400">
              Skilling, bosses y utilidades para tu cliente.
            </p>
          </div>

          <PluginsGrid />
        </section>

        <Pricing />
      </main>

      <footer className="border-t border-white/10 py-10">
        <div className="container-page flex flex-col items-center justify-between gap-4 text-sm text-slate-500 sm:flex-row">
          <p>© {new Date().getFullYear()} KatPlugins. No afiliado a Jagex.</p>
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
