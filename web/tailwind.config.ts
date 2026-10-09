import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./lib/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: "#DD1F2F",
          light: "#F26A76",
          dark: "#B01523",
          soft: "#FDE7E9",
        },
        ink: {
          950: "#0A0B0D",
          900: "#101114",
          850: "#16171B",
          800: "#1C1E23",
          700: "#26282E",
          600: "#33363D",
        },
      },
      fontFamily: {
        sans: ["var(--font-sans)", "system-ui", "sans-serif"],
        display: ["var(--font-display)", "var(--font-sans)", "system-ui", "sans-serif"],
      },
      boxShadow: {
        glow: "0 0 60px -18px rgba(221, 31, 47, 0.55)",
        card: "0 1px 0 0 rgba(255, 255, 255, 0.04) inset",
      },
    },
  },
  plugins: [],
};

export default config;
