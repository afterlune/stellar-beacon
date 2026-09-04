<template>
  <AgentRouteGate
    feature="publicChat"
    title="公开对话"
    description="对话只使用公开文章、分类、标签和生命体征，不具备写入后台或访问私有数据的能力。"
    eyebrow="PUBLIC CONVERSATION">
    <AgentWidget embedded />
  </AgentRouteGate>
</template>

<script lang="ts">
import { defineComponent } from 'vue'
import AgentRouteGate from '@/components/AgentRouteGate.vue'
import AgentWidget from '@/components/AgentWidget.vue'

export default defineComponent({
  name: 'AgentChat',
  components: { AgentRouteGate, AgentWidget }
})
</script>
