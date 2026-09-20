<template>
  <button
    type="button"
    class="follow-button"
    :class="{ 'is-following': followingState, 'is-compact': compact }"
    :disabled="disabled || busy || isSelf"
    @click="toggle">
    {{ isSelf ? '自己' : busy ? '处理中…' : followingState ? '已关注' : '关注' }}
  </button>
</template>

<script lang="ts">
import { computed, defineComponent, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '@/api/api'
import { useUserStore } from '@/stores/user'

export default defineComponent({
  name: 'FollowButton',
  props: {
    authorId: { type: Number, required: true },
    following: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
    compact: { type: Boolean, default: false }
  },
  emits: ['changed'],
  setup(props, { emit }) {
    const route = useRoute()
    const router = useRouter()
    const userStore = useUserStore()
    const followingState = ref(props.following)
    const isSelf = computed(() => Number(userStore.userInfo?.userInfoId || userStore.userInfo?.id || 0) === Number(props.authorId || 0))
    const busy = ref(false)
    watch(() => props.following, (value) => { followingState.value = value })
    const toggle = async () => {
      if (!userStore.userInfo) {
        await router.push({ path: route.path, query: { ...route.query, login: '1', redirect: route.fullPath } })
        return
      }
      busy.value = true
      try {
        const response = followingState.value ? await api.unfollowAuthor(props.authorId) : await api.followAuthor(props.authorId)
        if (!response?.data?.flag) throw new Error(response?.data?.message || '关注操作失败')
        followingState.value = !followingState.value
        emit('changed', followingState.value)
        ElMessage.success(followingState.value ? '已关注' : '已取消关注')
      } catch (reason: any) {
        ElMessage.error(reason?.response?.data?.message || reason?.message || '关注操作失败')
      } finally {
        busy.value = false
      }
    }
    return { followingState, isSelf, busy, toggle }
  }
})
</script>

<style scoped>
.follow-button { min-width: 86px; padding: 9px 16px; border: 1px solid var(--color-ob); border-radius: 999px; background: var(--color-ob); color: #081127; font-weight: 700; cursor: pointer; }
.follow-button.is-following { background: transparent; color: var(--color-ob); }
.follow-button:disabled { opacity: .55; cursor: wait; }
.follow-button.is-compact { min-width: 68px; padding: 6px 11px; font-size: 12px; }
</style>