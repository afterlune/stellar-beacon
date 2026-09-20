<template>
  <div>
    <div class="flex flex-col">
      <PageHeader :title="t('titles.talks')" :current="t('menu.talks')" />
      <div class="main-grid">
        <div class="relative space-y-5">
          <div class="talk-item" v-for="item in talks" :key="item.id" @click="toTalk(item.id)">
            <Avatar :url="item.avatar" />
            <div class="talk-info">
              <div class="user-nickname text-sm">
                {{ item.nickname }}
              </div>
              <div class="time">
                {{ t('settings.shared-on') }}
                {{ formatTime(item.createTime) }}
                <template v-if="item.isTop === 1">
                  <svg-icon icon-class="top" class="top-svg" /><span style="color: #f21835">置顶</span>
                </template>
                <svg-icon icon-class="message" class="message-svg" />{{
                  item.commentCount == null ? 0 : item.commentCount
                }}
              </div>
              <div class="talk-content" v-html="item.content" />
              <div class="talk-images" v-if="item.imgs">
                <img
                  class="talk-image"
                  v-for="(img, index) of item.imgs"
                  :key="index"
                  :src="safeTalkImageUrl(img)"
                  :alt="t('settings.talk-image', { index: Number(index) + 1 })"
                  @click.stop="handlePreview(img)" />
              </div>
            </div>
          </div>
          <Paginator
            :pageSize="pagination.size"
            :pageTotal="pagination.total"
            :page="pagination.current"
            @pageChange="pageChangeHanlder" />
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
import { defineComponent, onMounted, reactive, toRefs } from 'vue'
import { useI18n } from 'vue-i18n'
import { PageHeader } from '@/components/PageHeader'
import { Sidebar, Profile } from '../components/Sidebar'
import Paginator from '@/components/Paginator.vue'
import Avatar from '../components/Avatar.vue'
import { v3ImgPreviewFn } from 'v3-img-preview'
import { useRouter } from 'vue-router'
import api from '@/api/api'
import { safeTalkImageUrl } from '@/utils/image'

export default defineComponent({
  name: 'talkList',
  components: { PageHeader, Sidebar, Profile, Paginator, Avatar },
  setup() {
    const { t } = useI18n()
    const router = useRouter()
    const pagination = reactive({
      size: 7,
      total: 0,
      current: 1
    })
    const reactiveData = reactive({
      images: [] as any,
      talks: '' as any
    })
    onMounted(() => {
      fetchTalks()
    })
    const handlePreview = (index: any) => {
      v3ImgPreviewFn({ images: reactiveData.images, index: reactiveData.images.indexOf(index) })
    }
    const fetchTalks = () => {
      const params = {
        current: pagination.current,
        size: pagination.size
      }
      api.getTalks(params).then(({ data }) => {
        const page = data && data.data ? data.data : {}
        const records = Array.isArray(page.records) ? page.records : []
        reactiveData.images = []
        records.forEach((item: any) => {
          if (Array.isArray(item.imgs)) {
            item.imgs = item.imgs.map((image: any) => safeTalkImageUrl(image))
            reactiveData.images.push(...item.imgs)
          }
        })
        reactiveData.talks = records
        pagination.total = Number(page.count) || 0
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
    const pageChangeHanlder = (current: number) => {
      reactiveData.talks = ''
      toPageTop()
      pagination.current = current
      fetchTalks()
    }
    const toTalk = (id: any) => {
      router.push({ path: '/talks/' + id })
    }
    return {
      pagination,
      ...toRefs(reactiveData),
      formatTime,
      pageChangeHanlder,
      handlePreview,
      safeTalkImageUrl,
      toTalk,
      t
    }
  }
})
</script>

<style lang="scss" scoped>
.top-svg {
  margin-left: 5px;
}
.message-svg {
  margin-left: 5px;
  font-size: 15px;
}
.talk-item {
  display: flex;
  padding: 20px;
  border-radius: var(--radius-xl);
  background-color: var(--surface-1);
  border: none;
  box-shadow: inset 0 1px 0 var(--glass-edge), var(--shadow-card);
  cursor: pointer;
  transition: transform 200ms ease, box-shadow 200ms ease;
}
.talk-item:hover {
  transform: translateY(-2px);
  box-shadow: inset 0 1px 0 var(--glass-edge), 0 22px 48px -22px rgba(0, 0, 0, 0.55);
}
.talk-info {
  flex: 1;
  min-width: 0;
  margin-left: 12px;
}
.user-nickname {
  font-weight: 600;
  color: var(--text-bright);
}
.time {
  color: var(--text-dim);
  font-size: 12px;
}
.talk-content {
  margin-top: 8px;
  font-size: 15px;
  line-height: 1.75;
  color: var(--text-normal);
  white-space: pre-line;
  word-wrap: break-word;
  word-break: break-word;
}
/* el-col with :md="4" collapsed into odd shapes whenever an item had fewer than
   six images. Fixed-size tracks keep every thumbnail square whatever the count. */
.talk-images {
  display: grid;
  grid-template-columns: repeat(auto-fill, 118px);
  gap: 8px;
  margin-top: 12px;
}
.talk-image {
  width: 100%;
  height: auto;
  min-height: 0;
  border-radius: var(--radius-md);
  overflow: hidden;
  border: none;
}
.talk-image {
  display: block;
  object-fit: contain;
  object-position: center;
}
</style>
