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
          DEFAULT: "#FFD400",
          dark: "#E0B900",
          soft: "#FFF3B0",
        },
        ink: {
          950: "#08090C",
          900: "#0D0F14",
          850: "#12141B",
          800: "#171A22",
          700: "#22262F",
          600: "#31363F",
        },
      },
      fontFamily: {
        sans: ["var(--font-sans)", "system-ui", "sans-serif"],
      },
      boxShadow: {
        glow: "0 0 40px -10px rgba(255, 212, 0, 0.45)",
      },
    },
  },
  plugins: [],
};

export default config;
