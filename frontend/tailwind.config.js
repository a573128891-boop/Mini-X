/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#1d9bf0',
        dark: '#0f1419',
        darker: '#000000',
      },
    },
  },
  plugins: [],
}
