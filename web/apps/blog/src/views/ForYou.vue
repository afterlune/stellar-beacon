<template>
  <div class="for-you-page">
    <header class="for-you-hero">
      <p>PERSONAL DISCOVERY</p>
      <h1>为你推荐</h1>
      <span>从你的关注、话题订阅、收藏点赞和本机阅读记录中发现下一条公开信号。</span>
      <div>
        <router-link to="/topics">去话题广场</router-link>
        <router-link to="/authors">发现作者</router-link>
      </div>
    </header>
    <RecommendationPanel />
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue'
import RecommendationPanel from '@/components/RecommendationPanel.vue'

export default defineComponent({
  name: 'ForYou',
  components: { RecommendationPanel }
})
</script>

<style scoped>
.for-you-page { max-width: 1180px; margin: 0 auto; padding: 30px 0 96px; }
.for-you-hero { margin-bottom: 22px; padding: clamp(28px, 5vw, 52px); border: 1px solid var(--border-hairline); border-radius: 24px; background: radial-gradient(circle at 88% 10%, rgba(92, 118, 224, .22), transparent 35%), color-mix(in srgb, var(--background-primary-alt) 94%, transparent); }
.for-you-hero p { margin: 0 0 10px; color: var(--color-ob); font-size: 10px; letter-spacing: .2em; }
.for-you-hero h1 { margin: 0 0 12px; font-size: clamp(2.2rem, 5vw, 4rem); }
.for-you-hero > span { color: var(--text-ob-dim); font-size: 13px; line-height: 1.8; }
.for-you-hero > div { display: flex; gap: 10px; margin-top: 24px; }
.for-you-hero a { padding: 9px 15px; border: 1px solid var(--border-hairline); border-radius: 999px; color: inherit; font-size: 12px; text-decoration: none; }
.for-you-hero a:first-child { border-color: transparent; background: var(--color-ob); color: #081127; font-weight: 700; }
</style>
