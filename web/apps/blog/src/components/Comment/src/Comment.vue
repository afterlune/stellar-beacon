<template>
  <div class="surface-panel p-4 mt-8 lg:px-14 lg:py-10 rounded-2xl mb-8 lg:mb-0" id="comments">
    <SubTitle :title="'titles.comment'" />
    <CommentForm />
    <CommentList />
  </div>
</template>
<script lang="ts">
import { defineComponent } from 'vue'
import { SubTitle } from '@/components/Title'
import CommentForm from './CommentForm.vue'
import CommentList from './CommentList.vue'
export default defineComponent({
  name: 'Comment',
  components: { SubTitle, CommentForm, CommentList },
  setup() {
    return {}
  }
})
</script>
<style lang="scss">
.comment-focus {
  border-radius: 14px;
  animation: comment-focus-pulse 2.4s ease-out;
}
@keyframes comment-focus-pulse {
  0%, 24% { background: color-mix(in srgb, var(--color-ob) 16%, transparent); box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-ob) 45%, transparent); }
  100% { background: transparent; box-shadow: none; }
}
</style>