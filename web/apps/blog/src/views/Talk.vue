<template>
  <div>
    <div>
      <div class="flex flex-col">
        <div class="post-header">
          <h1 class="post-title text-white uppercase">{{ t('titles.talks') }}</h1>
        </div>
        <div class="main-grid">
          <div class="relative space-y-5">
            <div class="surface-panel flex p-4 lg:p-8 rounded-2xl mb-8 lg:mb-0">
              <Avatar v-if="talk.avatar" :url="talk.avatar" />
              <div class="talk-info">
                <div class="user-nickname text-sm">
                  {{ talk.nickname }}
                </div>
                <div v-if="talk.createTime" class="time">
                  {{ t('settings.shared-on') }}
                  {{ formatTime(talk.createTime) }}
                  <svg-icon icon-class="message" class="message-svg" />{{
                    talk.commentCount == null ? 0 : talk.commentCount
                  }}
                </div>
                <div class="talk-content" v-html="talk.content" />
                <el-row class="talk-images" v-if="talk.imgs">
                  <el-col :md="4" v-for="(img, index) of talk.imgs" :key="index">
                    <el-image
                      class="images-talks"
                      :src="safeTalkImageUrl(img)"
                      fit="contain"
                      @click.prevent="handlePreview(img)" />
                  </el-col>
                </el-row>
              </div>
            </div>
            <Comment />
          </div>
          <div class="col-span-1">
            <Sidebar>
              <Profile />
            </Sidebar>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, reactive, toRefs, provide, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Breadcrumb from '@/components/Breadcrumb.vue'
import { Sidebar, Profile } from '../components/Sidebar'
import { Comment } from '../components/Comment'
import Avatar from '../components/Avatar.vue'
import { useCommentStore } from '@/stores/comment'
import { v3ImgPreviewFn } from 'v3-img-preview'
import emitter from '@/utils/mitt'
import api from '@/api/api'
import { pageCount, pageRecords } from '@/utils/page'
import { safeTalkImageUrl } from '@/utils/image'

export default defineComponent({
  name: 'talks',
  components: { Breadcrumb, Sidebar, Profile, Comment, Avatar },
  setup() {
    const { t } = useI18n()
    const commentStore = useCommentStore()
    const route = useRoute()
    const router = useRouter()
    const reactiveData = reactive({
      talk: '' as any,
      comments: [] as any,
      haveMore: false as any,
      isReload: false as any,
      images: [] as any
    })
    const pageInfo = reactive({
      current: 1,
      size: 7
    })
    commentStore.type = 5
    onMounted(() => {
      toPageTop()
      fetchTalk()
      fetchComments()
    })
    provide(
      'comments',
      computed(() => reactiveData.comments)
    )
    provide(
      'haveMore',
      computed(() => reactiveData.haveMore)
    )
    emitter.on('talkFetchComment', () => {
      pageInfo.current = 1
      reactiveData.isReload = true
      fetchComments()
    })
    emitter.on('talkFetchReplies', (index) => {
      fetchReplies(index)
    })
    emitter.on('talkLoadMore', () => {
      fetchComments()
    })
    const handlePreview = (index: any) => {
      v3ImgPreviewFn({ images: reactiveData.images, index: reactiveData.images.indexOf(index) })
    }
    const fetchTalk = () => {
      api.getTalkById(route.params.talkId).then(({ data }) => {
        if (data.data === null) {
          router.push({ path: '/出错啦' })
          return
        }
        reactiveData.talk = data.data
        if (Array.isArray(reactiveData.talk.imgs)) {
          reactiveData.talk.imgs = reactiveData.talk.imgs.map((image: any) => safeTalkImageUrl(image))
          reactiveData.images.push(...reactiveData.talk.imgs)
        }
      })
    }
    const fetchComments = () => {
      const params = {
        type: 5,
        topicId: route.params.talkId,
        current: pageInfo.current,
        size: pageInfo.size
      }
      api.getComments(params).then(({ data }) => {
        const records = pageRecords(data)
        if (reactiveData.isReload) {
          reactiveData.comments = records
          reactiveData.isReload = false
        } else {
          reactiveData.comments.push(...records)
        }
        if (pageCount(data) <= reactiveData.comments.length) {
          reactiveData.haveMore = false
        } else {
          reactiveData.haveMore = true
        }
        pageInfo.current++
      })
    }
    const fetchReplies = (index: any) => {
      api.getRepliesByCommentId(reactiveData.comments[index].id).then(({ data }) => {
        reactiveData.comments[index].replyDTOs = data.data
      })
    }
    // One stable timestamp format across article, talk and comment surfaces.
    const formatTime = (data: any): string => {
      const date = new Date(data)
      if (Number.isNaN(date.getTime())) return ''
      const pad = (value: number): string => String(value).padStart(2, '0')
      return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
    }
    const toPageTop = () => {
      window.scrollTo({
        top: 0
      })
    }
    return {
      ...toRefs(reactiveData),
      handlePreview,
      safeTalkImageUrl,
      formatTime,
      t
    }
  }
})
</script>

<style lang="scss" scoped>
.message-svg {
  margin-left: 5px;
  font-size: 15px;
}
.el-card {
  background: var(--background-primary);
  border-radius: 10px;
  border: 0;
}
.talk-user-avatar {
  flex: 1;
}
.talk-info {
  flex: 1;
  margin-left: 10px;
}
.user-nickname {
  font-weight: 530;
}
.time {
  color: var(--text-dim);
  font-size: 13px;
  @media (min-width: 1280px) {
    margin-top: 4px;
  }
}
.talk-content {
  margin-top: 10px;
  font-size: 14px;
  line-height: 26px;
  white-space: pre-line;
  word-wrap: break-word;
  word-break: break-all;
}
.talk-images {
  margin-top: 8px;
}
.images-items {
  cursor: pointer;
  border-radius: 3px;
  margin-right: 5px;
}
.images-talks {
  display: block;
  width: 100%;
  height: auto;
}
.images-talks :deep(.el-image__inner) {
  display: block;
  width: 100%;
  height: auto;
  object-fit: contain;
  object-position: center;
}
</style>
