<template>
  <button type="button" class="comment-like" :class="{ 'is-active': active }" :disabled="busy" :aria-pressed="active" data-testid="comment-like" :data-comment-id="commentId" @click="toggle">
    <span aria-hidden="true">{{ active ? '♥' : '♡' }}</span>
    <span>{{ likeCount }}</span>
  </button>
</template>

<script lang="ts">
import { defineComponent, ref, watch } from 'vue'
import { notify } from '@/services/notifications'
import api from '@/api/api'
import { useUserStore } from '@/stores/user'

export default defineComponent({
  name: 'CommentLikeButton',
  props: {
    commentId: { type: Number, required: true },
    likeCount: { type: Number, default: 0 },
    liked: { type: Boolean, default: false }
  },
  emits: ['changed'],
  setup(props, { emit }) {
    const userStore = useUserStore()
    const active = ref(Boolean(props.liked))
    const count = ref(Number(props.likeCount || 0))
    const busy = ref(false)
    watch(() => props.liked, (value) => { active.value = Boolean(value) })
    watch(() => props.likeCount, (value) => { count.value = Number(value || 0) })
    const toggle = async () => {
      if (!userStore.userInfo) {
        userStore.userVisible = true
        return
      }
      busy.value = true
      try {
        const response = await api.setCommentReaction({ commentId: props.commentId, active: !active.value })
        if (!response?.data?.flag) throw new Error(response?.data?.message || '操作失败')
        active.value = Boolean(response.data.data?.active)
        count.value = Number(response.data.data?.likeCount || 0)
        emit('changed', { active: active.value, likeCount: count.value })
      } catch (reason: any) {
        notify.error(reason?.response?.data?.message || reason?.message || '操作失败')
      } finally {
        busy.value = false
      }
    }
    return { active, count, busy, toggle }
  }
})
</script>

<style scoped lang="scss">
.comment-like { display: inline-flex; align-items: center; gap: 4px; padding: 0; border: 0; background: transparent; color: var(--text-ob-dim); cursor: pointer; }
.comment-like:hover, .comment-like.is-active { color: var(--color-ob); }
.comment-like:disabled { opacity: .55; cursor: wait; }
</style>
