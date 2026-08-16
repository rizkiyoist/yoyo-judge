import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  // Relative asset paths so the built dist/ folder is relocatable — it can
  // be copied to any path/server (not just served from "/") and still load
  // its JS/CSS correctly.
  base: './',
  plugins: [vue()],
})
