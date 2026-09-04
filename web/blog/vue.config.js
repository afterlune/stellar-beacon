const path = require('path')
const { defineConfig } = require('@vue/cli-service')
function resolve(dir) {
  return path.join(__dirname, dir)
}
const apiTarget = process.env.VUE_APP_API_TARGET || 'https://localhost:7777'
module.exports = defineConfig({
  // Keep the rollback build separate from the canonical Vite dist directory.
  // Running the legacy build must not replace the artifact used by Caddy or
  // make the Vite performance budget report a different application.
  outputDir: 'dist-legacy',
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
        '@': resolve('src'),
        '@shared': resolve('../shared')
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
