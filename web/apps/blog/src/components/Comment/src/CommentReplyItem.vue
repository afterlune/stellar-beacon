<template>
  <div :id="`comment-${reply.id}`">
    <div class="flex space-x-3 xl:space-x-5">
      <Avatar :url="reply.avatar" />
      <div class="reply comment-bubble flex flex-col p-3 rounded-md relative">
        <p class="commentContent" v-html="commentContent.replaceAll('\n', '<br>')" />
        <div class="flex justify-between mt-2 text-xs text-ob-dim space-x-3 md:space-x-16">
          <span> {{ reply.nickname }} | {{ time }}</span>
          <div class="flex items-center gap-3">
            <CommentLikeButton
              :comment-id="Number(reply.id)"
              :like-count="Number(reply.likeCount || 0)"
              :liked="Boolean(reply.liked)"
              @changed="updateLike" />
            <span @click="clickOnSonReply" class="cursor-pointer reply-button">Reply</span>
            <button v-if="canModerate" type="button" class="reply-button" :disabled="busy" data-testid="comment-reply-delete-action" @click="removeReply">删除</button>
          </div>
        </div>
      </div>
    </div>
    <a href="" target="_blank"></a>
    <CommentReplyForm
      class="mt-5"
      v-show="show"
      :replyUserId="reply.userId"
      :initialContent="replyContent"
      @changeShow="changeShow" />
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, inject, reactive, ref, toRefs } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api/api'
import emitter from '@/utils/mitt'
import Avatar from '@/components/Avatar.vue'
import CommentReplyForm from './CommentReplyForm.vue'
import CommentLikeButton from './CommentLikeButton.vue'

export default defineComponent({
  components: {
    Avatar,
    CommentReplyForm,
    CommentLikeButton
  },
  props: ['reply', 'commentUserId'],
  setup(props) {
    const reply = computed(() => props.reply)
    const canModerate = inject<() => boolean>('canModerate', () => false)
    const readCollectionID = inject<() => number>('collectionId', () => 0)
    const rootCommentId = inject('parentId', 0)
    const busy = ref(false)
    const formatTime = (time: any): any => {
      let date = new Date(time)
      let year = date.getFullYear()
      let month = date.getMonth() + 1
      let day = date.getDate()
      return year + '-' + month + '-' + day
    }
    const reactiveData = reactive({
      replyContent: '' as any,
      time: formatTime(props.reply.createTime) as any,
      show: false as any
    })
    const clickOnSonReply = () => {
      reactiveData.replyContent = '@' + props.reply.nickname
      reactiveData.show = true
    }
    const changeShow = () => {
      reactiveData.show = false
    }
    const updateLike = (payload: { active: boolean; likeCount: number }) => {
      props.reply.likeCount = payload.likeCount
      props.reply.liked = payload.active
    }
    const commentContent = computed(() => {
      if (props.reply.replyUserId !== props.commentUserId) {
        return (
          `<a href="${props.reply.replyWebsite}" target="_blank" class="reply-link">@${props.reply.replyNickname}&nbsp</a>` +
          props.reply.commentContent
        )
      } else {
        return props.reply.commentContent
      }
    })
    const removeReply = async () => {
      const collectionID = Number(readCollectionID())
      if (!collectionID) return
      try {
        await ElMessageBox.confirm('删除后该回复将不再公开显示。', '删除回复', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' })
      } catch {
        return
      }
      busy.value = true
      try {
        const response = await api.deleteOwnedCollectionComment(collectionID, Number(reply.value.id))
        if (!response?.data?.flag) throw new Error(response?.data?.message || '操作失败')
        emitter.emit('collectionFetchReplies', Number(rootCommentId))
      } catch (reason: any) {
        ElMessage.error(reason?.response?.data?.message || reason?.message || '操作失败')
      } finally {
        busy.value = false
      }
    }
    return {
      ...toRefs(reactiveData),
      commentContent,
      clickOnSonReply,
      changeShow,
      updateLike,
      canModerate: computed(() => Boolean(canModerate())),
      busy,
      removeReply
    }
  }
})
</script>
<style lang="scss" scoped>
.reply::before {
  content: '';
  position: absolute;
  width: 0;
  height: 0;
  border-right: 8px solid var(--background-primary);
  border-top: 6px solid transparent;
  border-bottom: 6px solid transparent;
  left: -8px;
  top: 14px;
}
.reply {
  background: var(--background-primary);
}
.reply-button {
  color: var(--text-accent);
  cursor: pointer;
  border: 0;
  background: transparent;
  padding: 0;
  font-size: inherit;
}
.reply-button:disabled {
  opacity: .6;
  cursor: wait;
}
.commentContent {
  line-height: 26px;
  white-space: pre-line;
  word-wrap: break-word;
  word-break: break-all;
}
</style>
<style lang="scss">
.reply-link {
  color: var(--text-accent);
}
</style>
