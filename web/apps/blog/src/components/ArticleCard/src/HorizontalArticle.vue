<template>
  <div class="article-container">
    <span v-if="article.isTop" class="article-tag">
      <b>
        <svg-icon icon-class="pin" />
        {{ t('settings.pinned') }}
      </b>
    </span>
    <span v-else-if="article.isfeatured" class="article-tag">
      <b>
        <svg-icon icon-class="hot" />
        {{ t('settings.featured') }}
      </b>
    </span>
    <div class="feature-article">
      <div class="feature-thumbnail">
        <img v-if="article.articleCover" class="ob-hz-thumbnail" v-lazy="article.articleCover" />
        <img v-else class="ob-hz-thumbnail" src="@/assets/default-cover.jpg" />

      </div>
      <div class="feature-content">
        <span>
          <b v-if="article.categoryName">
            {{ article.categoryName }}
          </b>
          <ob-skeleton v-else tag="b" height="20px" width="35px" />
          <ul>
            <template v-if="article.tags && article.tags.length > 0">
              <li v-for="tag in article.tags" :key="tag.id">
                <em># {{ tag.tagName }}</em>
              </li>
            </template>
            <template v-else-if="article.tags && article.tags.length <= 0">
              <li>
                <em># {{ t('settings.default-tag') }}</em>
              </li>
            </template>
            <ob-skeleton v-else :count="2" tag="li" height="16px" width="35px" />
          </ul>
        </span>
        <h1 class="article-title" v-if="article.articleTitle" @click="toArticle" data-dia="article-link">
          <a>
            <span>{{ article.articleTitle }}</span>
            <svg-icon v-if="article.status == 2" icon-class="lock" class="lock-svg" />
          </a>
        </h1>
        <ob-skeleton v-else tag="h1" height="3rem" />
        <p v-if="article.articleContent">{{ article.articleContent }}</p>
        <ob-skeleton v-else tag="p" :count="4" height="20px" />
        <div class="article-footer" v-if="article && article.author">
          <div class="flex flex-row items-center">
            <img
              class="hover:opacity-50 cursor-pointer"
              :src="article.author.avatar || avatarPlaceholder"
              alt=""
              @error="handleImageError"
              @click="handleAuthorClick(article.author.website)" />
            <span class="text-ob-dim">
              <strong
                class="text-ob-normal pr-1.5 hover:text-ob hover:opacity-50 cursor-pointer"
                @click="handleAuthorClick(article.author.website)">
                {{ article.author.nickname }}
              </strong>
              {{ t('settings.shared-on') }} {{ t(`settings.months[${new Date(article.createTime).getMonth()}]`) }}
              {{ new Date(article.createTime).getDate() }}, {{ new Date(article.createTime).getFullYear() }}
            </span>
          </div>
        </div>
        <div class="article-footer" v-else>
          <div class="flex flex-row items-center mt-6">
            <ob-skeleton class="mr-2" height="28px" width="28px" :circle="true" />
            <span class="text-ob-dim mt-1">
              <ob-skeleton height="20px" width="150px" />
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, toRef, getCurrentInstance } from 'vue'
import { useUserStore } from '@/stores/user'
import { useRouter } from 'vue-router'
import { useArticleStore } from '@/stores/article'
import { useI18n } from 'vue-i18n'
import emitter from '@/utils/mitt'
import avatarPlaceholder from '@/assets/avatar-placeholder.svg'

export default defineComponent({
  name: 'HorizontalArticle',
  setup() {
    const proxy: any = getCurrentInstance()?.appContext.config.globalProperties
    const articleStore = useArticleStore()
    const userStore = useUserStore()
    const router = useRouter()
    const { t } = useI18n()
    const handleAuthorClick = (link: string) => {
      if (link === '') link = window.location.href
      window.open(link)
    }
    const handleImageError = (event: Event) => {
      const image = event.target as HTMLImageElement
      if (image.dataset.fallbackApplied === 'true') return
      image.dataset.fallbackApplied = 'true'
      image.src = avatarPlaceholder
    }
    const toArticle = () => {
      let isAccess = false
      userStore.accessArticles.forEach((item: any) => {
        if (item == articleStore.topArticle.id) {
          isAccess = true
        }
      })
      if (articleStore.topArticle.status == 2 && isAccess == false) {
        if (userStore.userInfo === '') {
          proxy.$notify({
            title: 'Warning',
            message: '该文章受密码保护,请登录后访问',
            type: 'warning'
          })
        } else {
          emitter.emit('changeArticlePasswordDialogVisible', articleStore.topArticle.id)
        }
      } else {
        router.push({ path: '/articles/' + articleStore.topArticle.id })
      }
    }
    return {

      article: toRef(articleStore.$state, 'topArticle'),
      handleAuthorClick,
      handleImageError,
      avatarPlaceholder,
      toArticle,
      t
    }
  }
})
</script>
<style lang="scss" scoped>
.article-title:hover {
  cursor: default;
}
</style>
