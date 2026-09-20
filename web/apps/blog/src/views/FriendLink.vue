<template>
  <div>
    <div class="flex flex-col">
      <PageHeader :title="t('titles.friends')" :current="t('menu.friends')" />
      <div class="main-grid">
        <div class="relative space-y-5">
          <div class="surface-panel p-4 lg:p-14 rounded-2xl mb-8 lg:mb-0">
            <el-row :gutter="36">
              <template v-for="link in links" :key="link.id">
                <el-col :span="8" :xs="{ span: 20, offset: 2 }" class="mb-3">
                  <el-card shadow="never" class="shadow-md">
                    <div class="block">
                    <img class="friend-avatar" :src="safeAvatarImageUrl(link.linkAvatar)" alt="" width="60" height="60" />
                    </div>
                    <div class="info">
                      <a :href="link.linkAddress" target="_blank">
                        <div class="link-name font-semibold">{{ link.linkName }}</div>
                      </a>
                      <div class="link-intro truncate">{{ link.linkIntro }}</div>
                    </div>
                  </el-card>
                </el-col>
              </template>
            </el-row>
          </div>
          <div
            class="post-html"
            v-html="
              `需要交换友链的可在下方留言💖<br><br>友链信息展示需要，你的信息格式要包含：名称、头像、链接、介绍`
            " />
          <section class="friend-apply">
            <h3 class="friend-apply__title">{{ t('friends.applyTitle') }}</h3>
            <p class="friend-apply__hint">{{ t('friends.applyHint') }}</p>
            <form class="friend-apply__form" @submit.prevent="submitApplication">
              <label>
                <span>{{ t('friends.applyName') }}</span>
                <input v-model="applyForm.linkName" type="text" maxlength="20" required />
              </label>
              <label>
                <span>{{ t('friends.applyAddress') }}</span>
                <input v-model="applyForm.linkAddress" type="url" placeholder="https://" required />
              </label>
              <label>
                <span>{{ t('friends.applyAvatar') }}</span>
                <input v-model="applyForm.linkAvatar" type="url" placeholder="https://" />
              </label>
              <label>
                <span>{{ t('friends.applyIntro') }}</span>
                <input v-model="applyForm.linkIntro" type="text" maxlength="100" />
              </label>
              <label>
                <span>{{ t('friends.applyEmail') }}</span>
                <input v-model="applyForm.email" type="email" />
              </label>
              <!-- Honeypot: hidden from readers, bots fill it. -->
              <input v-model="applyForm.website" class="friend-apply__trap" type="text" tabindex="-1" autocomplete="off" aria-hidden="true" />
              <button type="submit" :disabled="applyPending" data-testid="friend-apply-submit">{{ t('friends.applySubmit') }}</button>
              <span v-if="applyMessage" class="friend-apply__message">{{ applyMessage }}</span>
            </form>
          </section>
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
import { defineComponent, reactive, provide, computed, toRefs, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Sidebar, Profile } from '../components/Sidebar'
import { PageHeader } from '@/components/PageHeader'
import { Comment } from '../components/Comment'
import { useCommentStore } from '@/stores/comment'
import emitter from '@/utils/mitt'
import api from '@/api/api'
import { pageCount, pageRecords } from '@/utils/page'
import { safeAvatarImageUrl } from '@/utils/image'

export default defineComponent({
  name: 'FriendLink',
  components: { Sidebar, Profile, PageHeader, Comment },
  setup() {
    const { t } = useI18n()
    const commentStore = useCommentStore()
    const reactiveData = reactive({
      links: '' as any,
      comments: [] as any,
      haveMore: false as any,
      isReload: false as any
    })
    const pageInfo = reactive({
      current: 1,
      size: 7
    })
    commentStore.type = 4
    onMounted(() => {
      fetchLinks()
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
    emitter.on('friendLinkFetchComment', () => {
      pageInfo.current = 1
      reactiveData.isReload = true
      fetchComments()
    })
    emitter.on('friendLinkFetchReplies', (index) => {
      fetchReplies(index)
    })
    emitter.on('friendLinkLoadMore', () => {
      fetchComments()
    })
    const applyForm = reactive({ linkName: '', linkAddress: '', linkAvatar: '', linkIntro: '', email: '', website: '' })
    const applyPending = ref(false)
    const applyMessage = ref('')
    const submitApplication = () => {
      if (applyPending.value) return
      applyPending.value = true
      applyMessage.value = ''
      api
        .applyFriendLink({ ...applyForm })
        .then(({ data }: any) => {
          if (data?.code === 'OK') {
            applyMessage.value = t('friends.applyAccepted')
            applyForm.linkName = ''
            applyForm.linkAddress = ''
            applyForm.linkAvatar = ''
            applyForm.linkIntro = ''
            applyForm.email = ''
          } else {
            applyMessage.value = data?.message || t('friends.applyFailed')
          }
        })
        .catch(() => {
          applyMessage.value = t('friends.applyFailed')
        })
        .finally(() => {
          applyPending.value = false
        })
    }

    const fetchLinks = () => {
      api.getFriendLink().then(({ data }) => {
        reactiveData.links = Array.isArray(data.data) ? data.data : []
      })
    }
    const fetchComments = () => {
      const params = {
        type: 4,
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
      safeAvatarImageUrl,
      applyForm,
      applyPending,
      applyMessage,
      submitApplication,
      t
    }
  }
})
</script>

<style lang="scss" scoped>
.block {
  display: inline-block;
  width: 24%;
}
.info {
  display: inline-block;
  width: 76%;
  height: 100%;
}
.link-name {
  margin-left: 20px;
  margin-bottom: 5px;
  margin-top: 2px;
  color: var(--text-normal);
  font-size: large;
}
.link-intro {
  margin-left: 20px;
  margin-bottom: 1px;
  color: var(--text-normal);
}
.el-card {
  background: var(--background-primary);
  border-radius: 10px;
  border: 0;
}
.friend-avatar { width: 60px; height: 60px; border-radius: 50%; object-fit: cover; }
.info a { display: inline-flex; align-items: center; min-height: 44px; }
.friend-apply { margin: 2rem 0; padding: 1.25rem 1.35rem; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 24%, transparent); border-radius: 14px; }
.friend-apply__title { margin: 0 0 .35rem; font-family: var(--font-display); font-size: 1.15rem; }
.friend-apply__hint { margin: 0 0 1rem; font-size: .85rem; opacity: .7; }
.friend-apply__form { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: .75rem 1rem; align-items: end; }
.friend-apply__form label { display: flex; flex-direction: column; gap: .3rem; font-size: .82rem; opacity: .85; }
.friend-apply__form input { min-height: 44px; padding: .5rem .65rem; border: 1px solid color-mix(in srgb, var(--text-ob-dim) 28%, transparent); border-radius: 8px; background: transparent; color: inherit; }
.friend-apply__form button { min-height: 44px; padding: .55rem 1.1rem; border: none; border-radius: 999px; background: var(--color-ob); color: #fff; cursor: pointer; }
.friend-apply__form button:disabled { opacity: .6; cursor: progress; }
.friend-apply__trap { position: absolute; left: -9999px; width: 1px; height: 1px; opacity: 0; }
.friend-apply__message { grid-column: 1 / -1; font-size: .85rem; color: var(--color-ob); }
</style>