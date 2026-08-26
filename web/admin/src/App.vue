<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script>
import { generaMenu } from '@/assets/js/menu'
export default {
  created() {
    if (this.$store.state.userInfo != null) {
      const requestedPath = this.$route.path
      const target = requestedPath === '/login' ? '/' : this.$route.fullPath
      const replace = (path) => {
        const navigation = this.$router.replace({ path })
        if (navigation && typeof navigation.catch === 'function') {
          navigation.catch(() => {})
        }
      }
      generaMenu().then(() => {
        if (requestedPath === '/login') {
          replace('/')
          return
        }
        if (this.$route.matched.length === 0) {
          const navigation = this.$router.replace({ path: target })
          if (navigation && typeof navigation.then === 'function') {
            navigation.then(() => {
              if (this.$route.matched.length === 0) {
                replace('/')
              }
            }).catch(() => replace('/'))
          }
        }
      }).catch(() => {
        this.$store.commit('logout')
        this.$router.replace({ path: '/login' })
      })
    }
    this.axios.post('/api/report')
  }
}
</script>
