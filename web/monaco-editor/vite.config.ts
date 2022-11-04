import { defineConfig } from 'vite'

// https://vitejs.dev/config/
export default defineConfig({
  base: './',
  server: {
    port: 9804,
  },
  build: {
    outDir: '../public/code-editor',
    emptyOutDir: true,
  },
})
