<template><div ref="container" class="admin-echart" :style="{ height }" /></template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'

const props = withDefaults(defineProps<{ option: Record<string, unknown>; height?: string }>(), { height: '300px' })
const container = ref<HTMLElement | null>(null)
let chart: echarts.ECharts | null = null
let observer: ResizeObserver | null = null

function render(): void {
  if (!chart) return
  chart.setOption(props.option, true)
}

onMounted(async () => {
  await nextTick()
  if (!container.value) return
  chart = echarts.init(container.value)
  render()
  observer = new ResizeObserver(() => chart?.resize())
  observer.observe(container.value)
})

watch(() => props.option, render, { deep: true })

onBeforeUnmount(() => {
  observer?.disconnect()
  chart?.dispose()
  chart = null
})
</script>

<style scoped>
.admin-echart { width: 100%; min-height: 180px; }
</style>
