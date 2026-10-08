import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "KatPlugins — Plugins premium para RuneLite",
  description:
    "Plugins de alta calidad para RuneLite: skilling, bosses y QoL. Inicia sesión con Google y desbloquea tus plugins.",
  metadataBase: new URL("https://katplugins.pajau.cl"),
  openGraph: {
    title: "KatPlugins",
    description: "Plugins premium para RuneLite.",
    url: "https://katplugins.pajau.cl",
    siteName: "KatPlugins",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="es" suppressHydrationWarning>
      <body className="min-h-screen antialiased">{children}</body>
    </html>
  );
}
