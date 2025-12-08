import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const SNAPSHOT_PROXY_TARGET = process.env.VITE_SNAPSHOT_PROXY_TARGET

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  server: {
    proxy: {
      '/snapshots': {
        target: SNAPSHOT_PROXY_TARGET,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/snapshots/, ''),
      },
    },
  },
})
