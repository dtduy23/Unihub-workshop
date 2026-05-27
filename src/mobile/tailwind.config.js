/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./App.{js,jsx,ts,tsx}", "./src/**/*.{js,jsx,ts,tsx}"],
  theme: {
    extend: {
      colors: {
        primary: "#1E40AF", // Cobalt Blue
        background: "#F8FAFC", // Slate 50
        success: "#059669",
        error: "#E11D48",
        warning: "#D97706",
      },
    },
  },
  plugins: [],
}
