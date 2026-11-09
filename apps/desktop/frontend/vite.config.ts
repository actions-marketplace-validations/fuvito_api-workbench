import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': `${import.meta.dirname}/src`,
      '@wailsjs': `${import.meta.dirname}/wailsjs`,
    },
  },
  server: {
    // Wails WebView uses wails.localhost as the request origin in dev mode.
    allowedHosts: ['wails.localhost'],
  },
  build: {
    // Output to the Wails embed directory so Go can embed the built assets.
    outDir: '../wails/frontend/dist',
    emptyOutDir: true,
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/test/**', 'src/**/*.test.*', 'src/**/*.d.ts'],
    },
  },
})
