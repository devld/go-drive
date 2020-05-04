module.exports = {
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
  }
}
