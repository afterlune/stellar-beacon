<template>
  <AgentRouteGate
    :any-of="agentFeatures"
    title="Benetnasch Agent"
    description="一个只读取公开内容的数字生命入口。聊天、星河、梦境和电台都可以独立启用。"
    eyebrow="AURORA OF BENETNASCH">
    <section class="agent-home__links" aria-label="Agent 体验入口">
      <router-link v-if="features.publicChat" to="/agent/chat" class="agent-home__link">
        <strong>公开对话</strong>
        <span>和 Benetnasch 聊聊文章与标签</span>
      </router-link>
      <router-link v-if="features.galaxy" to="/galaxy" class="agent-home__link">
        <strong>内容星河</strong>
        <span>在二维投影中浏览公开文章</span>
      </router-link>
      <router-link v-if="features.dreams" to="/dreams" class="agent-home__link">
        <strong>梦境档案</strong>
        <span>阅读已经通过人工审核的梦境</span>
      </router-link>
    </section>

    <AgentVitals v-if="features.vitals" />
    <AgentGalaxy v-if="features.galaxy" />
    <AgentDreams v-if="features.dreams" />
    <AgentRadio v-if="features.radio" />
    <AgentVideos v-if="features.videos" />
  </AgentRouteGate>
</template>

<script lang="ts">
import { computed, defineComponent } from 'vue'
import { useAppStore } from '@/stores/app'
import AgentRouteGate from '@/components/AgentRouteGate.vue'
import type { PublicAgentFeature } from '@/components/AgentRouteGate.vue'
import AgentDreams from '@/components/AgentDreams.vue'
import AgentGalaxy from '@/components/AgentGalaxy.vue'
import AgentRadio from '@/components/AgentRadio.vue'
import AgentVideos from '@/components/AgentVideos.vue'
import AgentVitals from '@/components/AgentVitals.vue'

const agentFeatures: PublicAgentFeature[] = ['publicChat', 'vitals', 'galaxy', 'dreams', 'radio', 'videos']

export default defineComponent({
  name: 'AgentHome',
  components: {
    AgentRouteGate,
    AgentDreams,
    AgentGalaxy,
    AgentRadio,
    AgentVideos,
    AgentVitals
  },
  setup() {
    const appStore = useAppStore()
    return {
      agentFeatures,
      features: computed(() => appStore.agentFeatures)
    }
  }
})
</script>

<style lang="scss" scoped>
.agent-home__links {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
  margin-bottom: 28px;
}

.agent-home__link {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 92px;
  padding: 20px;
  border: 1px solid var(--bg-accent-05);
  border-radius: 14px;
  background: rgba(18, 27, 53, 0.46);
  color: var(--text-normal);
  text-decoration: none;
  transition: transform 180ms ease, border-color 180ms ease;
}

.agent-home__link:hover {
  border-color: var(--text-accent);
  transform: translateY(-2px);
}

.agent-home__link strong {
  color: var(--text-bright);
  font-size: 18px;
}

.agent-home__link span {
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1.6;
}

@media (prefers-reduced-motion: reduce) {
  .agent-home__link {
    transition: none;
  }

  .agent-home__link:hover {
    transform: none;
  }
}
</style>
