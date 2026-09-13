<template>
  <div class="sidebar-box">
    <SubTitle :title="'titles.recent_comment'" icon="quote" compact />
    <ul>
      <template v-if="comments.length > 0">
        <li
          class="recent-comment-item"
          v-for="comment in comments"
          :key="comment.id">
          <div class="flex-shrink-0 mr-2">
            <div class="rounded-full overflow-hidden w-9">
              <template v-if="comment.avatar">
                <img class="avatar-img" :src="comment.avatar" alt="" @error="handleImageError" />
              </template>
              <template v-else>
                <img class="avatar-img" :src="default" alt="" />
              </template>
            </div>
          </div>
          <div class="flex-1 text-xs comment">
            <div class="text-xs">
              <span class="text-ob pr-2">
                {{ comment.nickname }}
              </span>
              <p class="text-ob-dim">{{ comment.createTime }}</p>
            </div>
            <div class="text-xs text-ob-bright commentContent">
              {{ comment.commentContent }}
            </div>
          </div>
        </li>
      </template>
    </ul>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, toRef } from 'vue'
import { SubTitle } from '@/components/Title'
import { useCommentStore } from '@/stores/comment'
import { useI18n } from 'vue-i18n'
import api from '@/api/api'
import avatarPlaceholder from '@/assets/avatar-placeholder.svg'

export default defineComponent({
  name: 'RecentComment',
  components: { SubTitle },
  setup() {
    const commentStore = useCommentStore()
    const { t } = useI18n()
    onMounted(() => {
      initRecentComment()
    })
    const initRecentComment = () => {
      api.getTopSixComments().then(({ data }) => {
        if (data.data.length === 0) {
          commentStore.recentComment = []
        }
        data.data.forEach((itme: any) => {
          itme.createTime = formatTime(itme.createTime)
        })
        commentStore.recentComment = data.data
      })
    }
    const formatTime = (time: any): any => {
      let date = new Date(time)
      let year = date.getFullYear()
      let month = date.getMonth() + 1
      let day = date.getDate()
      return year + '-' + month + '-' + day
    }
    const handleImageError = (event: Event) => {
      const image = event.target as HTMLImageElement
      if (image.dataset.fallbackApplied === 'true') return
      image.dataset.fallbackApplied = 'true'
      image.src = avatarPlaceholder
    }
    return {
      comments: toRef(commentStore.$state, 'recentComment'),
      default: avatarPlaceholder,
      handleImageError,
      t
    }
  }
})
</script>

<style lang="scss" scoped>
.recent-comment-item {
  display: flex;
  flex-direction: row;
  align-items: center;
  padding: 10px 12px;
  margin-bottom: 6px;
  border-radius: var(--radius-md);
  background-color: var(--surface-2);
  border: none;
  transition: background-color 200ms ease;
}
.recent-comment-item:hover {
  background-color: var(--surface-hover);
}
.comment {
  width: 70%;
}
.commentContent {
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.avatar-img {
  transition-property: transform;
  transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
  transition-duration: 800ms;
  transform: rotate(-360deg);
}
.avatar-img:hover {
  transform: rotate(360deg);
}
</style>
