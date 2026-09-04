<template>
  <AgentRouteGate
    feature="dreams"
    title="梦境档案"
    description="这里的每一条梦境都经过人工审核；模型生成的候选不会直接出现在公开页面。"
    eyebrow="APPROVED DREAM ARCHIVE">
    <AgentDreams />
  </AgentRouteGate>
</template>

<script lang="ts">
import { defineComponent } from 'vue'
import AgentRouteGate from '@/components/AgentRouteGate.vue'
import AgentDreams from '@/components/AgentDreams.vue'

export default defineComponent({
  name: 'AgentDreamsView',
  components: { AgentRouteGate, AgentDreams }
})
</script>
