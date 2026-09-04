<template>
  <AgentRouteGate
    feature="galaxy"
    title="内容星河"
    description="只展示已经完成公开投影的文章坐标；原始向量、模型信息和私密内容不会发送到浏览器。"
    eyebrow="CONTENT CONSTELLATION">
    <AgentGalaxy />
  </AgentRouteGate>
</template>

<script lang="ts">
import { defineComponent } from 'vue'
import AgentRouteGate from '@/components/AgentRouteGate.vue'
import AgentGalaxy from '@/components/AgentGalaxy.vue'

export default defineComponent({
  name: 'AgentGalaxyView',
  components: { AgentRouteGate, AgentGalaxy }
})
</script>
