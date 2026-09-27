<template>
  <button
    type="button"
    class="subscribe-topic-button"
    :class="{ 'is-subscribed': subscribedState, 'is-compact': compact }"
    :disabled="disabled || busy"
    @click.stop.prevent="toggle">
    {{ busy ? '处理中…' : subscribedState ? '已订阅' : '订阅' }}
  </button>
</template>

<script lang="ts">
import { defineComponent, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { notify } from '@/services/notifications'
import api from '@/api/api'
import { useUserStore } from '@/stores/user'

export default defineComponent({
  name: 'SubscribeTopicButton',
  props: {
    topicType: { type: String, required: true },
    topicKey: { type: String, required: true },
    subscribed: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
    compact: { type: Boolean, default: false }
  },
  emits: ['changed'],
  setup(props, { emit }) {
    const route = useRoute()
    const router = useRouter()
    const userStore = useUserStore()
    const subscribedState = ref(props.subscribed)
    const busy = ref(false)
    watch(() => props.subscribed, (value) => { subscribedState.value = value })
    const toggle = async () => {
      if (!userStore.userInfo) {
        await router.push({ path: route.path, query: { ...route.query, login: '1', redirect: route.fullPath } })
        return
      }
      busy.value = true
      try {
        const response = subscribedState.value
          ? await api.unsubscribeTopic(props.topicType, props.topicKey)
          : await api.subscribeTopic(props.topicType, props.topicKey)
        if (!response?.data?.flag) throw new Error(response?.data?.message || '订阅操作失败')
        subscribedState.value = !subscribedState.value
        emit('changed', subscribedState.value)
        notify.success(subscribedState.value ? '已订阅话题' : '已取消订阅')
      } catch (reason: any) {
        notify.error(reason?.response?.data?.message || reason?.message || '订阅操作失败')
      } finally {
        busy.value = false
      }
    }
    return { subscribedState, busy, toggle }
  }
})
</script>

<style scoped>
.subscribe-topic-button { min-width: 76px; padding: 8px 14px; border: 1px solid var(--color-ob); border-radius: 999px; background: transparent; color: var(--color-ob); font-weight: 700; cursor: pointer; }
.subscribe-topic-button.is-subscribed { background: var(--color-ob); color: #081127; }
.subscribe-topic-button:disabled { opacity: .55; cursor: wait; }
.subscribe-topic-button.is-compact { min-width: 62px; padding: 6px 10px; font-size: 12px; }
</style>