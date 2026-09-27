<template>
  <button
    type="button"
    class="collection-subscribe-button"
    :class="{ 'is-subscribed': subscribed }"
    :disabled="busy"
    @click="toggle">
    {{ busy ? '处理中…' : subscribed ? '已订阅' : '订阅更新' }}
  </button>
</template>

<script lang="ts">
import { defineComponent, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { notify } from '@/services/notifications'
import api from '@/api/api'
import { useUserStore } from '@/stores/user'

export default defineComponent({
  name: 'CollectionSubscribeButton',
  props: {
    collectionId: { type: Number, required: true }
  },
  emits: ['changed'],
  setup(props, { emit }) {
    const route = useRoute()
    const router = useRouter()
    const userStore = useUserStore()
    const subscribed = ref(false)
    const busy = ref(false)
    const loadStatus = async () => {
      if (!userStore.token || !props.collectionId) return
      try {
        const response = await api.getCollectionSubscriptionStatus(props.collectionId)
        subscribed.value = Boolean(response?.data?.data?.subscribed)
      } catch {
        subscribed.value = false
      }
    }
    const toggle = async () => {
      if (!userStore.userInfo) {
        await router.push({ path: route.path, query: { ...route.query, login: '1', redirect: route.fullPath } })
        return
      }
      busy.value = true
      try {
        const response = subscribed.value ? await api.unsubscribeCollection(props.collectionId) : await api.subscribeCollection(props.collectionId)
        if (!response?.data?.flag) throw new Error(response?.data?.message || '订阅操作失败')
        subscribed.value = !subscribed.value
        emit('changed', subscribed.value)
        notify.success(subscribed.value ? '已订阅书单更新' : '已取消订阅')
      } catch (reason: any) {
        notify.error(reason?.response?.data?.message || reason?.message || '订阅操作失败')
      } finally {
        busy.value = false
      }
    }
    watch(() => [props.collectionId, userStore.token], () => void loadStatus(), { immediate: true })
    return { subscribed, busy, toggle }
  }
})
</script>

<style scoped>
.collection-subscribe-button { display: inline-flex; align-items: center; min-height: 36px; padding: 7px 13px; border: 1px solid var(--color-ob); border-radius: 999px; background: var(--color-ob); color: #081127; font-size: 12px; font-weight: 700; cursor: pointer; }
.collection-subscribe-button.is-subscribed { background: transparent; color: var(--color-ob); }
.collection-subscribe-button:disabled { opacity: .6; cursor: wait; }
</style>
