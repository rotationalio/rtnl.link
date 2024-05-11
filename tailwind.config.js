/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["pkg/rtnl/templates/**/*.html"],
  theme: {
    extend: {
      colors: {
        "lapis": "#1d65a6",
        "space-cadet": "#192e5b",
        "air-superiority": "#72a2c0",
        "ghost": "#f8f8ff",
        "dartmouth": "#00743f",
        "mint": "#46B47F",
        "orange": "#f2a104",
        "yellow": "#fdb40b",
        "sinopia": "#db3b00",
        "orioles": "#ec5012",
      },
    },
  },
  plugins: [require("daisyui")],
}

