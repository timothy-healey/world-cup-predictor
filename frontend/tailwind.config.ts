import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        primary: { DEFAULT: "#DD2680", soft: "#FBE4F0" },
        secondary: { DEFAULT: "#F0BA2C", soft: "#FBE9C8", ink: "#7A5910" },
        bg: "#FAF5EB",
        surface: { DEFAULT: "#FFFCF5", sunk: "#F3EBD8" },
        ink: {
          DEFAULT: "#1B0E12",
          2: "#4A3940",
          3: "#7A6770",
          4: "#BCB0B5",
        },
        correct: { DEFAULT: "#13633F", soft: "#D9F1E3" },
        wrong: { DEFAULT: "#C73848", soft: "#FBE0E2" },
        pending: { DEFAULT: "#3753B8", soft: "#DFE4FA" },
      },
      fontFamily: {
        display: ['"Big Shoulders Display"', "system-ui", "sans-serif"],
        body: ['"Inter"', "system-ui", "-apple-system", "sans-serif"],
        mono: ['ui-monospace', '"SF Mono"', '"DM Mono"', "monospace"],
      },
      fontSize: {
        "3xs": ["9px", { lineHeight: "1.4" }],
        "2xs": ["10px", { lineHeight: "1.4" }],
        xs: ["11px", { lineHeight: "1.4" }],
        sm: ["12px", { lineHeight: "1.5" }],
        "2sm": ["13px", { lineHeight: "1.4" }],
        base: ["14px", { lineHeight: "1.55" }],
        md: ["15px", { lineHeight: "1.5" }],
        lg: ["18px", { lineHeight: "1.35" }],
        xl: ["22px", { lineHeight: "1.2" }],
        "2xl": ["28px", { lineHeight: "1.1" }],
        "display-md": ["32px", { lineHeight: "1.05" }],
        "display-lg": ["34px", { lineHeight: "1.05" }],
        "3xl": ["40px", { lineHeight: "1.05" }],
        "display-xl": ["48px", { lineHeight: "1.0" }],
        hero: ["56px", { lineHeight: "1.0" }],
        "display-jumbo": ["88px", { lineHeight: "0.9" }],
      },
      letterSpacing: {
        display: "-0.01em",
        "display-tight": "-0.02em",
        "label-tight": "0.02em",
        "label-mid": "0.05em",
        label: "0.08em",
      },
      borderColor: {
        DEFAULT: "rgba(27,14,18,0.10)",
        strong: "rgba(27,14,18,0.20)",
      },
      borderRadius: {
        sm: "4px",
        DEFAULT: "6px",
        md: "8px",
        lg: "12px",
        xl: "16px",
        pill: "999px",
      },
      spacing: {
        px: "1px",
        0: "0",
        1: "4px",
        2: "8px",
        3: "12px",
        4: "16px",
        5: "20px",
        6: "24px",
        8: "32px",
        10: "40px",
        12: "48px",
        16: "64px",
        20: "80px",
      },
      transitionDuration: {
        fast: "120ms",
        DEFAULT: "150ms",
        slow: "240ms",
      },
      transitionTimingFunction: {
        "out-quart": "cubic-bezier(0.25, 1, 0.5, 1)",
        "out-expo": "cubic-bezier(0.16, 1, 0.3, 1)",
      },
      boxShadow: {
        hover: "0 2px 8px -2px rgba(27,14,18,0.08)",
        focus: "0 0 0 3px #FBE4F0",
      },
      keyframes: {
        pulse: {
          "0%": { boxShadow: "0 0 0 0 rgba(221,38,128,0.55)" },
          "100%": { boxShadow: "0 0 0 7px rgba(221,38,128,0)" },
        },
        wordIn: {
          "0%": { opacity: "0", transform: "translateY(2px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
      },
      animation: {
        pulse: "pulse 1.8s ease-out infinite",
        "word-in": "wordIn 240ms ease-out",
      },
    },
  },
  plugins: [],
} satisfies Config;
