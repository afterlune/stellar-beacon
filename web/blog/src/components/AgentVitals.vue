<template>
  <section v-if="visible" :class="['agent-vitals', emotionClass]" aria-label="Benetnasch 生命体征">
    <div class="agent-vitals__heading">
      <div>
        <span class="agent-vitals__eyebrow">BENETNASCH SIGNAL</span>
        <h2>星图状态</h2>
      </div>
      <span v-if="vitals" class="agent-vitals__phase">{{ phaseLabel }}</span>
    </div>
    <p v-if="loading" class="agent-vitals__hint">正在读取公开状态……</p>
    <p v-else-if="errorMessage" class="agent-vitals__hint">{{ errorMessage }}</p>
    <template v-else-if="vitals">
      <p class="agent-vitals__status">{{ statusLabel }} · {{ emotionLabel }}</p>
      <div v-if="emotionItems.length" class="agent-vitals__emotions" aria-label="情绪展示信号">
        <div v-for="item in emotionItems" :key="item.key" class="agent-vitals__emotion">
          <span>{{ item.label }}</span>
          <strong>{{ item.percent }}%</strong>
        </div>
      </div>
      <dl class="agent-vitals__grid">
        <div>
          <dt>公开文章</dt>
          <dd>{{ vitals.articleCount }}</dd>
        </div>
        <div>
          <dt>访客</dt>
          <dd>{{ vitals.uniqueVisitorCount }}</dd>
        </div>
        <div>
          <dt>近 24 小时更新</dt>
          <dd>{{ vitals.recentContentCount }}</dd>
        </div>
        <div>
          <dt>累计浏览</dt>
          <dd>{{ vitals.viewCount }}</dd>
        </div>
      </dl>
      <small class="agent-vitals__time">{{ localTimeLabel }}</small>
    </template>
  </section>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, ref } from 'vue'
import api from '@/api/api'

interface AgentVitalsData {
  status: string
  lifeStage: string
  emotion: string
  emotionScores?: Record<string, number>
  phase: string
  localTime: string
  articleCount: number
  recentContentCount: number
  uniqueVisitorCount: number
  viewCount: number
}

export default defineComponent({
  name: 'AgentVitals',
  setup() {
    const visible = ref(true)
    const loading = ref(true)
    const errorMessage = ref('')
    const vitals = ref<AgentVitalsData | null>(null)

    onMounted(async () => {
      try {
        const { data } = await api.getAgentVitals()
        if (!data?.flag || !data.data) {
          visible.value = false
          return
        }
        vitals.value = data.data as AgentVitalsData
      } catch (_error) {
        visible.value = false
      } finally {
        loading.value = false
      }
    })

    return {
      visible,
      loading,
      errorMessage,
      vitals,
      phaseLabel: computed(() => phaseLabel(vitals.value?.phase)),
      statusLabel: computed(() => statusLabel(vitals.value?.status)),
      localTimeLabel: computed(() => formatLocalTime(vitals.value?.localTime)),
      emotionLabel: computed(() => emotionLabel(vitals.value?.emotion)),
      emotionClass: computed(() => emotionClass(vitals.value?.emotion)),
      emotionItems: computed(() => emotionItems(vitals.value?.emotionScores))
    }
  }
})

function phaseLabel(phase?: string): string {
  switch (phase) {
    case 'dusk':
      return '黄昏'
    case 'night':
      return '深夜'
    case 'awake':
      return '清醒'
    default:
      return '稳定'
  }
}

function statusLabel(status?: string): string {
  switch (status) {
    case 'dusk':
      return '正在回望'
    case 'resting':
      return '安静休息'
    case 'awake':
      return '清醒在线'
    default:
      return '状态稳定'
  }
}

function emotionLabel(emotion?: string): string {
  switch (emotion) {
    case 'toxic':
      return '毒舌'
    case 'gentle':
      return '温柔'
    case 'melancholic':
      return '忧郁'
    case 'reflective':
      return '回望'
    case 'quiet':
      return '安静'
    default:
      return '平静'
  }
}

function emotionClass(emotion?: string): string {
  switch (emotion) {
    case 'toxic':
      return 'agent-vitals--toxic'
    case 'gentle':
      return 'agent-vitals--gentle'
    case 'melancholic':
      return 'agent-vitals--melancholic'
    default:
      return 'agent-vitals--neutral'
  }
}

function emotionItems(scores?: Record<string, number>): Array<{ key: string; label: string; percent: number }> {
  if (!scores) return []
  return [
    ['toxic', '毒舌'],
    ['gentle', '温柔'],
    ['melancholic', '忧郁']
  ].map(([key, label]) => ({
    key,
    label,
    percent: Math.round(Math.max(0, Math.min(1, Number(scores[key] || 0))) * 100)
  }))
}

function formatLocalTime(value?: string): string {
  if (!value) return ''
  const match = value.match(/T(\d{2}:\d{2})/)
  return match ? `服务端时间 ${match[1]}` : ''
}
</script>

<style lang="scss" scoped>
.agent-vitals {
  margin-bottom: 2rem;
  padding: 1rem 1.25rem;
  border: 1px solid var(--bg-accent-55);
  border-radius: 1rem;
  background: var(--background-secondary);
  box-shadow: var(--accent-shadow);
}

.agent-vitals__heading,
.agent-vitals__grid {
  display: grid;
  gap: 0.75rem;
}

.agent-vitals__heading {
  grid-template-columns: 1fr auto;
  align-items: start;
}

.agent-vitals__eyebrow {
  color: var(--text-accent);
  font-size: 0.68rem;
  letter-spacing: 0.12em;
}

.agent-vitals h2 {
  margin: 0.2rem 0 0;
  color: var(--text-bright);
  font-size: 1.25rem;
}

.agent-vitals__phase {
  border-radius: 999px;
  padding: 0.25rem 0.6rem;
  background: var(--bg-accent-05);
  color: var(--text-accent);
  font-size: 0.78rem;
}

.agent-vitals__status,
.agent-vitals__hint {
  margin: 0.85rem 0;
  color: var(--text-normal);
}

.agent-vitals__emotions {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.5rem;
  margin-bottom: 0.85rem;
}

.agent-vitals__emotion {
  display: flex;
  justify-content: space-between;
  gap: 0.5rem;
  padding: 0.45rem 0.55rem;
  border-radius: 0.55rem;
  background: var(--bg-accent-05);
  color: var(--text-dim);
  font-size: 0.72rem;
}

.agent-vitals__emotion strong {
  color: var(--text-bright);
}

.agent-vitals--toxic {
  border-color: color-mix(in srgb, var(--danger) 55%, var(--bg-accent-55));
}

.agent-vitals--gentle {
  border-color: color-mix(in srgb, var(--ok) 55%, var(--bg-accent-55));
}

.agent-vitals--melancholic {
  border-color: color-mix(in srgb, #8ebbe8 60%, var(--bg-accent-55));
}

.agent-vitals__hint,
.agent-vitals__time {
  color: var(--text-dim);
}

.agent-vitals__grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin: 0;
}

.agent-vitals__grid div {
  padding: 0.55rem;
  border-radius: 0.65rem;
  background: var(--bg-accent-05);
}

.agent-vitals dt {
  color: var(--text-dim);
  font-size: 0.72rem;
}

.agent-vitals dd {
  margin: 0.2rem 0 0;
  color: var(--text-bright);
  font-size: 1.1rem;
  font-weight: 700;
}

@media (max-width: 640px) {
  .agent-vitals__emotions {
    grid-template-columns: 1fr;
  }

  .agent-vitals__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
