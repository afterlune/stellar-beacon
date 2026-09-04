<template>
  <main class="agent-route-page" :aria-busy="!featuresReady">
    <section v-if="!featuresReady" class="agent-route-state" role="status">
      <span class="agent-route-state__eyebrow">BENETNASCH SIGNAL</span>
      <h1>正在确认公开状态</h1>
      <p>请稍候，正在读取当前体验开关。</p>
    </section>

    <section v-else-if="!enabled" class="agent-route-state" data-agent-route-state="disabled" role="status">
      <span class="agent-route-state__eyebrow">BENETNASCH SIGNAL</span>
      <h1>Agent 功能暂未公开</h1>
      <p>{{ disabledMessage }}</p>
      <router-link class="agent-route-state__back" to="/">返回首页</router-link>
    </section>

    <template v-else>
      <header class="agent-route-header">
        <div>
          <span class="agent-route-header__eyebrow">{{ eyebrow }}</span>
          <h1>{{ title }}</h1>
          <p>{{ description }}</p>
        </div>
        <router-link class="agent-route-header__back" to="/">返回首页</router-link>
      </header>
      <slot />
    </template>
  </main>
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from 'vue'
import { useAppStore } from '@/stores/app'

export type PublicAgentFeature =
  | 'publicChat'
  | 'vitals'
  | 'galaxy'
  | 'dreams'
  | 'capsules'
  | 'radio'
  | 'videos'

export default defineComponent({
  name: 'AgentRouteGate',
  props: {
    feature: {
      type: String as PropType<PublicAgentFeature>,
      default: undefined
    },
    anyOf: {
      type: Array as PropType<PublicAgentFeature[]>,
      default: () => []
    },
    title: {
      type: String,
      required: true
    },
    description: {
      type: String,
      required: true
    },
    eyebrow: {
      type: String,
      default: 'BENETNASCH AGENT'
    },
    disabledMessage: {
      type: String,
      default: '这个入口正在等待管理员逐步启用。'
    }
  },
  setup(props) {
    const appStore = useAppStore()
    const enabled = computed(() => {
      if (props.feature) return appStore.agentFeatures[props.feature] === true
      return props.anyOf.some((feature) => appStore.agentFeatures[feature] === true)
    })

    return {
      enabled,
      featuresReady: computed(() => appStore.agentFeaturesReady),
      title: computed(() => props.title),
      description: computed(() => props.description),
      eyebrow: computed(() => props.eyebrow),
      disabledMessage: computed(() => props.disabledMessage)
    }
  }
})
</script>

<style lang="scss" scoped>
.agent-route-page {
  position: relative;
  z-index: 3;
  max-width: 1280px;
  margin: 0 auto;
  padding: 72px 0 32px;
  color: var(--text-normal);
}

.agent-route-header,
.agent-route-state {
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 20px;
  background: rgba(18, 27, 53, 0.62);
  box-shadow: var(--accent-shadow);
}

.agent-route-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 28px;
  padding: 30px 34px;
}

.agent-route-header__eyebrow,
.agent-route-state__eyebrow {
  color: var(--text-accent);
  font-size: 11px;
  letter-spacing: 3px;
}

.agent-route-header h1,
.agent-route-state h1 {
  margin: 8px 0 10px;
  color: var(--text-bright);
  font-size: clamp(2rem, 5vw, 3.4rem);
}

.agent-route-header p,
.agent-route-state p {
  margin: 0;
  max-width: 660px;
  line-height: 1.8;
  color: var(--text-muted);
}

.agent-route-header__back,
.agent-route-state__back {
  flex: 0 0 auto;
  border: 1px solid var(--text-accent);
  border-radius: 999px;
  padding: 8px 16px;
  color: var(--text-bright);
  text-decoration: none;
  white-space: nowrap;
}

.agent-route-state {
  padding: 72px 34px;
  text-align: center;
}

.agent-route-state p {
  margin: 0 auto;
}

.agent-route-state__back {
  display: inline-block;
  margin-top: 24px;
}

@media (max-width: 640px) {
  .agent-route-page {
    padding-top: 36px;
  }

  .agent-route-header {
    flex-direction: column;
    padding: 24px;
  }

  .agent-route-state {
    padding: 54px 24px;
  }
}
</style>
