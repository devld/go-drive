import { defineConfig, loadEnv, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { createHtmlPlugin } from 'vite-plugin-html'
import { visualizer } from 'rollup-plugin-visualizer'
import { readFile } from 'node:fs/promises'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  base: './',
  server: {
    port: 9803,
    proxy: {
      '/api': {
        target: 'http://localhost:8089',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
  define: {},

  plugins: [
    vue(),

    // vite-plugin-html rewrites HTML requests to the SPA entry before Vite's
    // public directory middleware can serve them.
    servePublicHtml(),

    createHtmlPlugin({
      minify: true,
      inject: {
        data: {
          ...loadEnv(mode, __dirname),
          mode,
        },
      },
    }),
  ],
  resolve: {
    alias: {
      '@': path.join(__dirname, 'src'),
    },
  },
  build: {
    cssCodeSplit: false,
    rollupOptions: {
      plugins: [visualizer()],
      output: {
        chunkFileNames: 'assets/[name]-[hash].js',
        entryFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash][extname]',
        manualChunks: (id) => {
          if (id.includes('node_modules')) {
            if (id.includes('vue') || id.includes('pinia')) return 'vue'
          }
        },
      },
    },
  },
}))

const servePublicHtml = (): Plugin => ({
  name: 'serve-public-html',
  enforce: 'pre' as const,
  configureServer(server) {
    if (!server.config.publicDir) return
    const publicDir = path.resolve(server.config.publicDir)

    server.middlewares.use(async (req, res, next) => {
      if (req.method !== 'GET' && req.method !== 'HEAD') return next()

      let pathname: string
      try {
        pathname = decodeURIComponent(
          new URL(req.url ?? '/', 'http://localhost').pathname
        )
      } catch {
        return next()
      }
      if (!pathname.endsWith('.html') || pathname.includes('\0')) return next()

      const file = path.resolve(publicDir, `.${pathname}`)
      if (!file.startsWith(`${publicDir}${path.sep}`)) return next()

      try {
        const html = await readFile(file)
        res.statusCode = 200
        res.setHeader('Content-Type', 'text/html; charset=utf-8')
        res.setHeader('Cache-Control', 'no-cache')
        res.end(req.method === 'HEAD' ? undefined : html)
      } catch (error) {
        if ((error as NodeJS.ErrnoException).code === 'ENOENT') return next()
        next(error)
      }
    })
  },
})
