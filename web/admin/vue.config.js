const { defineConfig } = require('@vue/cli-service')
const path = require('path')
function resolve(dir) {
  return path.join(__dirname, dir)
}
const apiTarget = process.env.VUE_APP_API_TARGET || 'https://localhost:7777'
module.exports = defineConfig({
  transpileDependencies: true,
  productionSourceMap: false,
  devServer: {
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
        secure: false,
        pathRewrite: {
          '^/api': ''
        }
      }
    }
  },
  chainWebpack: (config) => {
    config.resolve.alias.set('@', resolve('src'))
  }
})
