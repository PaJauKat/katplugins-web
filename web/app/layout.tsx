import type { Metadata } from "next";
import { Inter, Sora } from "next/font/google";
import "./globals.css";

const sans = Inter({
  subsets: ["latin"],
  variable: "--font-sans",
  display: "swap",
});

const display = Sora({
  subsets: ["latin"],
  variable: "--font-display",
  display: "swap",
});

export const metadata: Metadata = {
  title: "KatPlugins — Plugins para RuneLite",
  description:
    "Plugins de alta calidad para RuneLite: skilling, bosses y QoL. Inicia sesión con Google y desbloquea tus plugins.",
  metadataBase: new URL("https://katplugins.pajau.cl"),
  openGraph: {
    title: "KatPlugins",
    description: "Plugins de alta calidad para RuneLite.",
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
      <body
        className={`${sans.variable} ${display.variable} min-h-screen font-sans antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
