import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const snapshotProxyTarget = env.VITE_SNAPSHOT_PROXY_TARGET

  return {
    define: {
      global: 'globalThis',
    },
    plugins: [
      react({
        babel: {
          plugins: ['babel-plugin-react-compiler'],
        },
      }),
      tailwindcss(),
    ],
    server: {
      proxy: snapshotProxyTarget
        ? {
          '/snapshots': {
            target: snapshotProxyTarget,
            changeOrigin: true,
            rewrite: (path) => path.replace(/^\/snapshots/, ''),
          },
        }
        : undefined,
    },
  }
})
