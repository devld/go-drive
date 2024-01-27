import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { injectHtml, minifyHtml } from 'vite-plugin-html'
import vueI18n from '@intlify/vite-plugin-vue-i18n'
import { visualizer } from 'rollup-plugin-visualizer'

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
    injectHtml({
      data: {
        ...loadEnv(mode, __dirname),
        mode,
      },
    }),
    minifyHtml(),
    vueI18n({
      include: path.resolve(__dirname, './src/i18n/lang/**'),
      runtimeOnly: false,
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
        chunkFileNames(chunkInfo) {
          if (chunkInfo.facadeModuleId && chunkInfo.name === 'index') {
            const dir = path.dirname(chunkInfo.facadeModuleId)
            return `${path.basename(path.dirname(dir))}-${path.basename(
              dir
            )}-[hash].js`.replace('dist-', '')
          }
          return '[name]-[hash].js'
        },
      },
    },
  },
}))
