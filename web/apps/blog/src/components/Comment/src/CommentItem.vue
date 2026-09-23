<template>
  <div :id="`comment-${comment.id}`" class="mt-5 max-w-full" :class="{ 'is-hidden': isHidden }">
    <div class="flex space-x-3 xl:space-x-5">
      <Avatar :url="comment.avatar" />
      <div class="max-w-full-calc space-y-5">
        <div class="comment-bubble p-4 rounded-md relative reply" style="width: fit-content">
          <p class="commentContent" v-html="comment.commentContent.replaceAll('\n', '<br>')" />
          <div class="flex justify-between mt-3 text-xs text-ob-dim space-x-3 md:space-x-16">
            <span>{{ comment.nickname }} | {{ time }}</span>
            <div class="flex items-center gap-3">
              <span v-if="isPinned" class="pin-badge" data-testid="comment-pinned">置顶</span>
              <CommentLikeButton v-if="!isHidden"
                :comment-id="Number(comment.id)"
                :like-count="Number(comment.likeCount || 0)"
                :liked="Boolean(comment.liked)"
                @changed="updateLike" />
              <span v-if="!isHidden" @click="clickOnReply" class="cursor-pointer reply-button">Reply</span>
              <CommentGovernanceActions :comment="comment" />
              <template v-if="canModerate && !isHidden">
                <button type="button" class="reply-button" :disabled="busy" data-testid="comment-pin-action" @click="togglePin">{{ isPinned ? '取消置顶' : '置顶' }}</button>
                <button type="button" class="reply-button" :disabled="busy" data-testid="comment-delete-action" @click="removeComment">删除</button>
              </template>
            </div>
          </div>
        </div>
        <CommentReplyForm
          v-if="!isHidden && show"
          :replyUserId="comment.userId"
          :initialContent="replyContent"
          @changeShow="changeShow" />
        <transition-group name="fade">
          <CommentReplyItem
            v-for="reply in comment.replyDTOs"
            :key="reply.id"
            :reply="reply"
            :commentUserId="comment.userId" />
        </transition-group>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, inject, reactive, ref, toRefs, provide } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import Avatar from '@/components/Avatar.vue'
import CommentReplyItem from './CommentReplyItem.vue'
import CommentReplyForm from './CommentReplyForm.vue'
import CommentLikeButton from './CommentLikeButton.vue'
import CommentGovernanceActions from './CommentGovernanceActions.vue'
import api from '@/api/api'
import emitter from '@/utils/mitt'

export default defineComponent({
  components: {
    Avatar,
    CommentReplyItem,
    CommentReplyForm,
    CommentLikeButton,
    CommentGovernanceActions
  },
  props: ['comment', 'index'],
  setup(props) {
    const comment: any = props.comment
    provide('parentId', comment.id)
    provide('index', () => Number(props.index))
    const canModerate = inject<() => boolean>('canModerate', () => false)
    const readCollectionID = inject<() => number>('collectionId', () => 0)
    const busy = ref(false)
    const moderate = async (action: (collectionID: number) => Promise<any>) => {
      const collectionID = Number(readCollectionID())
      if (!collectionID) return
      busy.value = true
      try {
        const response = await action(collectionID)
        if (!response?.data?.flag) throw new Error(response?.data?.message || '操作失败')
        emitter.emit('collectionFetchComment')
      } catch (reason: any) {
        ElMessage.error(reason?.response?.data?.message || reason?.message || '操作失败')
      } finally {
        busy.value = false
      }
    }
    const formatTime = (time: any): any => {
      let date = new Date(time)
      let year = date.getFullYear()
      let month = date.getMonth() + 1
      let day = date.getDate()
      return year + '-' + month + '-' + day
    }
    const reactiveData = reactive({
      replyContent: '' as any,
      time: formatTime(props.comment.createTime) as any,
      show: false as any
    })
    const changeShow = () => {
      reactiveData.show = false
    }
    const clickOnReply = () => {
      reactiveData.replyContent = 'add reply...'
      reactiveData.show = true
    }
    const updateLike = (payload: { active: boolean; likeCount: number }) => {
      comment.likeCount = payload.likeCount
      comment.liked = payload.active
    }
    const isPinned = computed(() => Boolean(props.comment.isTop))
    const isHidden = computed(() => Number(props.comment.isDelete || 0) === 1)
    const togglePin = () => moderate((collectionID) => api.pinCollectionComment(collectionID, Number(comment.id), !isPinned.value))
    const removeComment = () => moderate(async (collectionID) => {
      await ElMessageBox.confirm('删除后该评论及其回复将不再公开显示。', '删除评论', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' })
      return api.deleteOwnedCollectionComment(collectionID, Number(comment.id))
    })
    return {
      ...toRefs(reactiveData),
      clickOnReply,
      changeShow,
      updateLike,
      canModerate: computed(() => Boolean(canModerate())),
      busy,
      isPinned,
      isHidden,
      togglePin,
      removeComment
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
.pin-badge {
  border-radius: 999px;
  padding: 0 8px;
  color: #fff;
  background: var(--main-gradient);
}
.commentContent {
  line-height: 26px;
  white-space: pre-line;
  word-wrap: break-word;
  word-break: break-all;
}
.is-hidden { opacity: .78; }
</style>
