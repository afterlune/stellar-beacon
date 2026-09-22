<template>
  <div class="flex space-x-3 xl:space-x-5">
    <Avatar :url="avatar" />
    <div class="comment flex flex-col flex-wrap-reverse w-full max-w-full-calc">
      <textarea
        v-model="commentContent"
        id="comment-content"
        name="commentContent"
        class="w-full shadow-md rounded-md p-4 focus:outline-none input"
        placeholder="Add comment..."
        cols="30"
        rows="5" />
      <div class="justify-between" style="text-align: right">
        <button
          @click="saveComment"
          id="submit-button"
          class="mt-5 w-32 text-white p-2 rounded-lg shadow-lg transition transform hover:scale-105 flex float-right">
          <span class="text-center flex-grow commit">Add Comment</span>
        </button>
      </div>
      <div class="w-full border-b-2 mt-6 wire"></div>
    </div>
  </div>
</template>
<script lang="ts">
import { defineComponent, toRefs, reactive, getCurrentInstance, computed } from 'vue'
import Avatar from '@/components/Avatar.vue'
import { SubTitle } from '@/components/Title'
import { useUserStore } from '@/stores/user'
import { useCommentStore } from '@/stores/comment'
import { useAppStore } from '@/stores/app'
import api from '@/api/api'
import emitter from '@/utils/mitt'

export default defineComponent({
  name: 'CommentItem',
  components: { SubTitle, Avatar },
  setup() {
    const proxy: any = getCurrentInstance()?.appContext.config.globalProperties
    const userStore = useUserStore()
    const commentStore = useCommentStore()
    const appStore = useAppStore()
    const reactiveData = reactive({
      commentContent: '' as any
    })
    const saveComment = () => {
      if (userStore.userInfo === '') {
        proxy.$notify({
          title: '提示',
          message: '请登录后评论',
          type: 'warning'
        })
        return
      }
      if (reactiveData.commentContent.trim() == '') {
        proxy.$notify({
          title: '提示',
          message: '评论不能为空',
          type: 'warning'
        })
        return
      }
      const params: any = {
        commentContent: reactiveData.commentContent,
        type: commentStore.type,
        topicId: commentStore.topicId
      }
      api.saveComment(params).then(({ data }) => {
        if (data.flag) {
          fetchComments()
          let isCommentReview = appStore.websiteConfig.isCommentReview
          if (isCommentReview) {
            proxy.$notify({
              title: '提示',
              message: '评论成功,正在审核中',
              type: 'warning'
            })
          } else {
            proxy.$notify({
              title: '成功',
              message: '评论成功',
              type: 'success'
            })
          }
          reactiveData.commentContent = ''
        } else {
          proxy.$notify({
            title: '错误',
            message: data.message,
            type: 'error'
          })
        }
      })
    }
    const fetchComments = () => {
      switch (commentStore.type) {
        case 1:
          emitter.emit('articleFetchComment')
          break
        case 2:
          emitter.emit('messageFetchComment')
          break
        case 3:
          emitter.emit('aboutFetchComment')
          break
        case 4:
          emitter.emit('friendLinkFetchComment')
          break
        case 5:
          emitter.emit('talkFetchComment')
          break
        case 6:
          emitter.emit('collectionFetchComment')
          break
      }
    }
    return {
      ...toRefs(reactiveData),
      avatar: computed(() => userStore.userInfo.avatar),
      saveComment
    }
  }
})
</script>

<style lang="scss" scoped>
.input {
  background: var(--surface-2);
  border: 1px solid var(--border-hairline);
  resize: none;
}
#submit-button {
  outline: none;
  background: var(--main-gradient);
}
/* Use the theme-aware border color for the divider. */
.wire {
  border-color: var(--border-hairline);
}
</style>
