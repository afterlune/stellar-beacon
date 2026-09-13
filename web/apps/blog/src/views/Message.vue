<template>
  <div>
    <div class="flex flex-col">
      <PageHeader :title="t('titles.message')" :current="t('menu.message')" />
      <div class="main-grid">
        <div class="relative">
          <div class="post-html" v-html="`这是一个留言版<br><br>欢迎大家前来留言💖`" />
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
</template>
<script lang="ts">
import { defineComponent, onMounted, reactive, toRefs, computed, provide } from 'vue'
import { useI18n } from 'vue-i18n'
import { Sidebar, Profile } from '../components/Sidebar'
import { PageHeader } from '@/components/PageHeader'
import { Comment } from '../components/Comment'
import { useCommentStore } from '@/stores/comment'
import api from '@/api/api'
import emitter from '@/utils/mitt'
import { pageCount, pageRecords } from '@/utils/page'

export default defineComponent({
  name: 'Message',
  components: { PageHeader, Comment, Sidebar, Profile },
  setup() {
    const { t } = useI18n()
    const commentStore = useCommentStore()
    const reactiveData = reactive({
      comments: [] as any,
      haveMore: false as any,
      isReload: false as any
    })
    const pageInfo = reactive({
      current: 1,
      size: 7
    })
    commentStore.type = 2
    onMounted(() => {
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
    emitter.on('messageFetchComment', () => {
      pageInfo.current = 1
      reactiveData.isReload = true
      fetchComments()
    })
    emitter.on('messageFetchReplies', (index) => {
      fetchReplies(index)
    })
    emitter.on('messageLoadMore', () => {
      fetchComments()
    })
    const fetchComments = () => {
      const params = {
        type: 2,
        topicId: null,
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
    return {
      ...toRefs(reactiveData),
      t
    }
  }
})
</script>
