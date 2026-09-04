<template>
  <section v-if="visible" class="agent-radio" aria-label="Benetnasch 文本电台">
    <div class="agent-radio__heading">
      <div>
        <span class="agent-radio__eyebrow">BENETNASCH RADIO</span>
        <h2>夜航电台</h2>
      </div>
      <span v-if="episode" class="agent-radio__phase">{{ phaseLabel }}</span>
    </div>
    <p v-if="loading" class="agent-radio__hint">正在准备节目……</p>
    <p v-else-if="errorMessage" class="agent-radio__hint">{{ errorMessage }}</p>
    <template v-else-if="episode">
      <h3>{{ episode.title }}</h3>
      <p class="agent-radio__script">{{ episode.script }}</p>
      <div class="agent-radio__footer">
        <button
          v-if="episode.tts && ttsEnabled && speechSupported"
          type="button"
          :aria-pressed="speaking"
          @click="toggleSpeech">
          {{ speaking ? '停止朗读' : '朗读节目' }}
        </button>
        <span v-else-if="episode.tts && ttsEnabled" class="agent-radio__hint">当前浏览器不支持语音朗读。</span>
        <router-link v-if="episode.sourceArticleId" :to="`/articles/${episode.sourceArticleId}`">
          查看本期文章
        </router-link>
      </div>
    </template>
  </section>
</template>

<script lang="ts">
import { computed, defineComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import api from '@/api/api'
import { useAppStore } from '@/stores/app'

interface RadioEpisode {
  title: string
  script: string
  phase: string
  tts: boolean
  sourceArticleId?: number
}

export default defineComponent({
  name: 'AgentRadio',
  setup() {
    const appStore = useAppStore()
    const visible = ref(true)
    const loading = ref(true)
    const errorMessage = ref('')
    const episode = ref<RadioEpisode | null>(null)
    const speechSupported = ref(false)
    const speaking = ref(false)

    onMounted(async () => {
      speechSupported.value = typeof window !== 'undefined' && 'speechSynthesis' in window
      try {
        const { data } = await api.getRadio()
        if (!data?.flag || !data.data?.records?.length) {
          visible.value = false
          return
        }
        episode.value = data.data.records[0] as RadioEpisode
      } catch (_error) {
        visible.value = false
      } finally {
        loading.value = false
      }
    })

    const toggleSpeech = () => {
      if (!episode.value || !speechSupported.value || !appStore.agentFeatures.ttsEnabled) return
      if (speaking.value) {
        window.speechSynthesis.cancel()
        speaking.value = false
        return
      }
      window.speechSynthesis.cancel()
      const utterance = new SpeechSynthesisUtterance(episode.value.script)
      utterance.lang = 'zh-CN'
      utterance.rate = 0.95
      utterance.onend = () => {
        speaking.value = false
      }
      utterance.onerror = () => {
        speaking.value = false
      }
      speaking.value = true
      window.speechSynthesis.speak(utterance)
    }

    onBeforeUnmount(() => {
      if (speechSupported.value) window.speechSynthesis.cancel()
    })

    return {
      visible,
      loading,
      errorMessage,
      episode,
      ttsEnabled: computed(() => appStore.agentFeatures.ttsEnabled),
      speechSupported: computed(() => speechSupported.value && appStore.agentFeatures.ttsEnabled),
      speaking,
      toggleSpeech,
      phaseLabel: computed(() => phaseLabel(episode.value?.phase))
    }
  }
})

function phaseLabel(phase?: string): string {
  switch (phase) {
    case 'dusk':
      return '黄昏'
    case 'night':
      return '深夜'
    default:
      return '清醒'
  }
}
</script>

<style lang="scss" scoped>
.agent-radio {
  margin: 0 auto 40px;
  padding: 28px 32px;
  max-width: 1280px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 18px;
  background: rgba(18, 27, 53, 0.62);
  color: var(--text-normal);
}

.agent-radio__heading,
.agent-radio__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.agent-radio__eyebrow {
  color: var(--text-accent);
  font-size: 11px;
  letter-spacing: 3px;
}

.agent-radio h2,
.agent-radio h3 {
  margin: 6px 0 0;
  color: var(--text-bright);
}

.agent-radio h3 {
  margin-top: 24px;
  font-size: 18px;
}

.agent-radio__phase {
  color: var(--text-accent);
  font-size: 13px;
}

.agent-radio__script {
  margin: 16px 0 24px;
  line-height: 1.9;
  white-space: pre-line;
}

.agent-radio__hint {
  color: var(--text-muted);
}

.agent-radio button,
.agent-radio a {
  color: var(--text-bright);
  border: 1px solid var(--text-accent);
  border-radius: 999px;
  padding: 7px 14px;
  background: transparent;
  text-decoration: none;
  cursor: pointer;
}

@media (max-width: 640px) {
  .agent-radio {
    padding: 22px;
  }

  .agent-radio__footer {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
