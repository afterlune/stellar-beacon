<template>
  <section class="agent-dreams" aria-label="Benetnasch 公开梦境">
    <div class="agent-dreams__heading">
      <div>
        <span class="agent-dreams__eyebrow">APPROVED DREAM ARCHIVE</span>
        <h2>梦境档案</h2>
      </div>
      <button type="button" :disabled="loading" @click="loadPage(1, true)">刷新</button>
    </div>

    <p v-if="loading && dreams.length === 0" class="agent-dreams__hint" role="status">正在翻阅梦境档案……</p>
    <p v-else-if="errorMessage" class="agent-dreams__hint" role="alert">{{ errorMessage }}</p>
    <p v-else-if="dreams.length === 0" class="agent-dreams__hint">暂时还没有公开梦境。</p>
    <div v-else class="agent-dreams__grid">
      <article v-for="dream in dreams" :key="dream.id" class="agent-dreams__card">
        <img
          class="agent-dreams__image"
          :src="imageURL(dream)"
          :alt="dream.title || '梦境占位图'"
          loading="lazy"
          @error="replaceImage" />
        <div class="agent-dreams__body">
          <h3>{{ dream.title || '未命名梦境' }}</h3>
          <p>{{ dream.content }}</p>
          <time :datetime="dream.updatedAt || dream.createdAt">{{ formatTime(dream.updatedAt || dream.createdAt) }}</time>
        </div>
      </article>
    </div>

    <footer class="agent-dreams__footer">
      <span v-if="dreams.length > 0">已读取 {{ dreams.length }} 个梦境</span>
      <button v-if="hasMore" type="button" :disabled="loading" @click="loadPage(page + 1, false)">加载更多</button>
      <span v-if="loading && dreams.length > 0">同步中……</span>
    </footer>
  </section>
</template>

<script lang="ts">
import { defineComponent, onMounted, ref } from 'vue'
import api from '@/api/api'

interface DreamRecord {
  id: string
  title: string
  content: string
  imageUrl?: string
  imageStatus?: string
  createdAt?: string
  updatedAt?: string
}

interface DreamPage {
  records?: DreamRecord[]
  hasMore?: boolean
}

const PAGE_SIZE = 20
const PLACEHOLDER = '/dream-placeholder.svg'

export default defineComponent({
  name: 'AgentDreams',
  setup() {
    const dreams = ref<DreamRecord[]>([])
    const page = ref(1)
    const hasMore = ref(false)
    const loading = ref(false)
    const errorMessage = ref('')

    const loadPage = async (current: number, replace: boolean) => {
      if (loading.value) return
      loading.value = true
      errorMessage.value = ''
      try {
        const { data } = await api.getDreams({ current, size: PAGE_SIZE })
        if (!data?.flag) {
          errorMessage.value = data?.message || '梦境暂时离线。'
          return
        }
        const result = (data.data || {}) as DreamPage
        const incoming = Array.isArray(result.records) ? result.records : []
        if (replace) {
          dreams.value = incoming
        } else {
          const known = new Map(dreams.value.map((dream) => [dream.id, dream]))
          incoming.forEach((dream) => known.set(dream.id, dream))
          dreams.value = Array.from(known.values())
        }
        page.value = current
        hasMore.value = result.hasMore === true
      } catch (_error) {
        errorMessage.value = '梦境暂时离线，请稍后再试。'
      } finally {
        loading.value = false
      }
    }

    const imageURL = (dream: DreamRecord) => {
      const value = String(dream.imageUrl || '')
      if (dream.imageStatus !== 'ready' && dream.imageStatus !== 'placeholder') return PLACEHOLDER
      if (value.startsWith('/') || value.startsWith('https://')) return value
      return PLACEHOLDER
    }

    const replaceImage = (event: Event) => {
      const image = event.currentTarget
      if (!(image instanceof HTMLImageElement) || image.src.endsWith(PLACEHOLDER)) return
      image.src = PLACEHOLDER
    }

    onMounted(() => {
      void loadPage(1, true)
    })

    return {
      dreams,
      page,
      hasMore,
      loading,
      errorMessage,
      loadPage,
      imageURL,
      replaceImage,
      formatTime: (value?: string) => formatTime(value)
    }
  }
})

function formatTime(value?: string): string {
  if (!value) return '时间未知'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '时间未知'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium' }).format(date)
}
</script>

<style lang="scss" scoped>
.agent-dreams {
  padding: 28px 32px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 18px;
  background: rgba(18, 27, 53, 0.62);
  color: var(--text-normal);
}

.agent-dreams__heading,
.agent-dreams__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.agent-dreams__eyebrow {
  color: var(--text-accent);
  font-size: 11px;
  letter-spacing: 3px;
}

.agent-dreams h2,
.agent-dreams h3 {
  margin: 6px 0 0;
  color: var(--text-bright);
}

.agent-dreams__heading button,
.agent-dreams__footer button {
  border: 1px solid var(--text-accent);
  border-radius: 999px;
  padding: 7px 14px;
  color: var(--text-bright);
  background: transparent;
  cursor: pointer;
}

.agent-dreams button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.agent-dreams__hint {
  margin: 24px 0 0;
  color: var(--text-muted);
}

.agent-dreams__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 20px;
  margin-top: 24px;
}

.agent-dreams__card {
  overflow: hidden;
  border: 1px solid var(--bg-accent-05);
  border-radius: 14px;
  background: var(--background-primary);
}

.agent-dreams__image {
  display: block;
  width: 100%;
  aspect-ratio: 16 / 9;
  object-fit: cover;
  background: var(--background-secondary);
}

.agent-dreams__body {
  padding: 18px;
}

.agent-dreams__body h3 {
  margin: 0 0 10px;
  font-size: 18px;
}

.agent-dreams__body p {
  min-height: 5.5em;
  margin: 0;
  line-height: 1.8;
  white-space: pre-line;
  overflow-wrap: anywhere;
}

.agent-dreams__body time {
  display: block;
  margin-top: 16px;
  color: var(--text-dim);
  font-size: 12px;
}

.agent-dreams__footer {
  margin-top: 24px;
  color: var(--text-muted);
  font-size: 13px;
}

@media (max-width: 640px) {
  .agent-dreams {
    padding: 22px;
  }

  .agent-dreams__heading,
  .agent-dreams__footer {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
