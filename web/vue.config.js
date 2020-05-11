const CopyWebpackPlugin = require('copy-webpack-plugin')
const path = require('path')

module.exports = {
  publicPath: './',
  productionSourceMap: false,

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
  }
}
