import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { injectHtml, minifyHtml } from 'vite-plugin-html'
import vueI18n from '@intlify/vite-plugin-vue-i18n'
import { visualizer } from 'rollup-plugin-visualizer'
import fs from 'fs'

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
  define: {
    JS_DECLARATIONS: JSON.stringify(readEnvDeclarations()),
  },

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
    },
  },
}))

function readEnvDeclarations() {
  const dir = '../docs/scripts'

  const global = fs
    .readFileSync(path.join(dir, 'global.d.ts'))
    .toString('utf-8')

  const readDeclarations = (dir: string) => {
    return fs
      .readdirSync(dir)
      .filter(
        (name) =>
          name.endsWith('.d.ts') && fs.statSync(path.join(dir, name)).isFile()
      )
      .map((name) => ({
        name: name.substring(0, name.length - 5), // .d.ts
        content: fs.readFileSync(path.join(dir, name)).toString('utf-8'),
      }))
      .reduce((a, c) => {
        a[c.name] = c.content
        return a
      }, {} as Record<string, string>)
  }

  const libs = readDeclarations(path.join(dir, 'libs'))
  const env = readDeclarations(path.join(dir, 'env'))

  return { global, env, libs }
}
