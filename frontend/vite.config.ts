import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig(({ command, mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  // Matches router.go's basePath() default — the shared prefix the backend
  // serves everything under (see ../orkestrator-v2 for the pattern this is
  // based on).
  const basePath = (env.VITE_BASE_PATH || '/yoyojudge').replace(/\/$/, '')

  return {
    // The dev server always runs at root (simplest for local dev); only
    // the production build is served under the shared basePath prefix.
    // Absolute paths matter here: a *relative* base ('./') breaks Vue
    // Router's history-mode deep links on a hard refresh (asset URLs would
    // resolve relative to the current nested route, not the real root/
    // prefix) — always use an absolute (optionally prefixed) path instead.
    base: command === 'build' ? `${basePath}/` : '/',
    plugins: [vue()],
    server: {
      proxy: {
        // Forward API calls made against the relative default (see
        // src/api/http.ts) to the backend during local dev.
        [basePath]: {
          target: 'http://localhost:5000',
          changeOrigin: true,
        },
      },
    },
  }
})
