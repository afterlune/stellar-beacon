<template>
  <section v-if="visible" class="agent-videos" aria-label="Benetnasch 视频">
    <div class="agent-videos__heading">
      <div>
        <span class="agent-videos__eyebrow">BENETNASCH MOVIES</span>
        <h2>影像档案</h2>
      </div>
      <button type="button" :disabled="loading" @click="loadVideos">刷新</button>
    </div>
    <p v-if="loading && videos.length === 0" class="agent-videos__hint">正在读取影像档案……</p>
    <p v-else-if="errorMessage" class="agent-videos__hint">{{ errorMessage }}</p>
    <p v-else-if="videos.length === 0" class="agent-videos__hint">暂时还没有公开视频。</p>
    <div v-else class="agent-videos__grid">
      <article v-for="video in videos" :key="video.id" class="agent-videos__item">
        <video
          v-if="video.source === 'local' && safeLocalURL(video.url)"
          :src="video.url"
          controls
          preload="metadata"
          :aria-label="video.title"
        />
        <iframe
          v-else-if="video.source === 'external' && safeExternalURL(video.embedUrl)"
          :src="video.embedUrl"
          :title="video.title"
          loading="lazy"
          referrerpolicy="no-referrer"
          sandbox="allow-scripts allow-same-origin"
          allowfullscreen
        />
        <p v-else class="agent-videos__hint">视频地址未通过安全校验。</p>
        <h3>{{ video.title }}</h3>
        <p v-if="video.description">{{ video.description }}</p>
      </article>
    </div>
  </section>
</template>

<script lang="ts">
import { defineComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import api from '@/api/api'

interface VideoRecord {
  id: string
  title: string
  description?: string
  source: string
  url: string
  embedUrl?: string
}

const CSP_META_ID = 'benetnasch-video-frame-policy'

export default defineComponent({
  name: 'AgentVideos',
  setup() {
    const visible = ref(true)
    const loading = ref(false)
    const errorMessage = ref('')
    const videos = ref<VideoRecord[]>([])
    const frameOrigins = ref<string[]>([])

    const loadVideos = async () => {
      loading.value = true
      errorMessage.value = ''
      try {
        const { data } = await api.getVideos({ current: 1, size: 20 })
        if (!data?.flag || !data.data) {
          visible.value = false
          return
        }
        videos.value = (data.data.records || []) as VideoRecord[]
        frameOrigins.value = Array.isArray(data.data.frameOrigins) ? data.data.frameOrigins : []
        installFramePolicy(frameOrigins.value)
      } catch (_error) {
        visible.value = false
      } finally {
        loading.value = false
      }
    }

    const safeLocalURL = (value: string) => isHTTPURL(value)
    const safeExternalURL = (value?: string) => {
      if (!value || !isHTTPURL(value)) return false
      try {
        return frameOrigins.value.includes(new URL(value).origin)
      } catch (_error) {
        return false
      }
    }

    onMounted(loadVideos)
    onBeforeUnmount(() => {
      const meta = document.getElementById(CSP_META_ID)
      if (meta) meta.remove()
    })

    return {
      visible,
      loading,
      errorMessage,
      videos,
      loadVideos,
      safeLocalURL,
      safeExternalURL
    }
  }
})

function isHTTPURL(value: string): boolean {
  try {
    const parsed = new URL(value)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
  } catch (_error) {
    return false
  }
}

function installFramePolicy(origins: string[]) {
  const safeOrigins = origins.filter((origin) => {
    try {
      const parsed = new URL(origin)
      return parsed.protocol === 'https:' && parsed.origin === origin
    } catch (_error) {
      return false
    }
  })
  let meta = document.getElementById(CSP_META_ID) as HTMLMetaElement | null
  if (!meta) {
    meta = document.createElement('meta')
    meta.id = CSP_META_ID
    meta.httpEquiv = 'Content-Security-Policy'
    document.head.appendChild(meta)
  }
  meta.content = `frame-src 'self' ${safeOrigins.join(' ')}; object-src 'none'; base-uri 'none'`
}
</script>

<style lang="scss" scoped>
.agent-videos {
  margin: 0 auto 40px;
  padding: 28px 32px;
  max-width: 1280px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 18px;
  background: rgba(18, 27, 53, 0.62);
}

.agent-videos__heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.agent-videos__eyebrow {
  color: var(--text-accent);
  font-size: 11px;
  letter-spacing: 3px;
}

.agent-videos h2,
.agent-videos h3 {
  margin: 6px 0 0;
  color: var(--text-bright);
}

.agent-videos__heading button {
  padding: 7px 14px;
  color: var(--text-bright);
  border: 1px solid var(--text-accent);
  border-radius: 999px;
  background: transparent;
  cursor: pointer;
}

.agent-videos__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 24px;
  margin-top: 24px;
}

.agent-videos__item video,
.agent-videos__item iframe {
  display: block;
  width: 100%;
  aspect-ratio: 16 / 9;
  border: 0;
  border-radius: 12px;
  background: #090d1b;
}

.agent-videos__item h3 {
  font-size: 17px;
}

.agent-videos__item p,
.agent-videos__hint {
  color: var(--text-muted);
}

@media (max-width: 640px) {
  .agent-videos {
    padding: 22px;
  }
}
</style>
