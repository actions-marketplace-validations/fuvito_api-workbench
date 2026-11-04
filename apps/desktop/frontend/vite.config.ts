import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': `${import.meta.dirname}/src`,
      '@wailsjs': `${import.meta.dirname}/wailsjs`,
    },
  },
  build: {
    // Output to the Wails embed directory so Go can embed the built assets.
    outDir: '../wails/frontend/dist',
    emptyOutDir: true,
  },
})
