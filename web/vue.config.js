const CopyWebpackPlugin = require('copy-webpack-plugin')
const path = require('path')

module.exports = {
  publicPath: './',
  productionSourceMap: false,

  css: {
    loaderOptions: {
      sass: {
        prependData: '@import "~@/styles/themes/include.scss";'
      }
    }
  },

  pages: {
    index: {
      entry: 'src/main.js',
      template: 'public/index.html',
      filename: 'index.html',
      title: process.env.VUE_APP_SITE_TITLE
    }
  },

  devServer: {
    proxy: {
      '/api': {
        target: 'http://localhost:8089',
        changeOrigin: true,
        pathRewrite: {
          '^/api': ''
        }
      }
    }
  },

  configureWebpack: {
    plugins: [
      new CopyWebpackPlugin([{
        from: path.resolve(__dirname, 'node_modules/codemirror/mode/*/*'),
        to: path.resolve(__dirname, 'dist/static/codemirror/'),
        context: path.resolve(__dirname, 'node_modules/codemirror/')
      }])
    ]
  },
  pwa: {
    name: process.env.VUE_APP_SITE_TITLE,
    themeColor: '#f2f5fa',
    msTileColor: '#2b5797',
    iconPaths: {
      favicon32: 'static/icon/favicon-32x32.png',
      favicon16: 'static/icon/favicon-16x16.png',
      appleTouchIcon: 'static/icon/apple-touch-icon.png',
      maskIcon: 'static/icon/safari-pinned-tab.svg',
      msTileImage: 'static/icon/android-chrome-512x512.png'
    },
    manifestOptions: {
      icons: [
        { src: 'static/icon/android-chrome-192x192.png', sizes: '192x192', type: 'image/png' },
        { src: 'static/icon/android-chrome-512x512.png', sizes: '512x512', type: 'image/png' }
      ],
      background_color: '#fff'
    }
  }
}
