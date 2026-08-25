const path = require('path')
const { defineConfig } = require('@vue/cli-service')
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
  configureWebpack: {
    resolve: {
      alias: {
        '@': resolve('src')
      }
    }
  },
  chainWebpack: (config) => {
    config.resolve.alias.set('vue-i18n', 'vue-i18n/dist/vue-i18n.cjs.js')
    config.module.rule('svg').exclude.add(resolve('src/icons')).end()
    config.module
      .rule('icons')
      .test(/\.svg$/)
      .include.add(resolve('src/icons'))
      .end()
      .use('svg-sprite-loader')
      .loader('svg-sprite-loader')
      .options({
        symbolId: 'icon-[name]'
      })
      .end()
  }
})
