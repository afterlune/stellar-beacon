<template>
  <div ref="container" class="admin-echart" :style="{ height }" />
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as echarts from 'echarts'

import { useThemeStore } from '@/stores/theme'

const props = withDefaults(defineProps<{ option: Record<string, unknown>; height?: string }>(), {
  height: '300px'
})

const theme = useThemeStore()
const container = ref<HTMLElement | null>(null)

let chart: echarts.ECharts | null = null
let observer: ResizeObserver | null = null

/**
 * ECharts logs a console error when initialized into a zero-sized container,
 * which would break the "console must stay clean" test contract. Defer the
 * first init until the element actually has a box.
 */
function ensureChart(): echarts.ECharts | null {
  if (chart) return chart
  const element = container.value
  if (!element || element.clientWidth === 0 || element.clientHeight === 0) return null
  chart = echarts.init(element, theme.theme === 'dark' ? 'dark' : undefined)
  chart.setOption(props.option, true)
  return chart
}

function render(): void {
  const instance = ensureChart()
  if (!instance) return
  instance.setOption(props.option, true)
}

onMounted(async () => {
  await nextTick()
  const element = container.value
  if (!element) return
  render()
  observer = new ResizeObserver(() => {
    if (!chart) render()
    else chart.resize()
  })
  observer.observe(element)
})

watch(() => props.option, render, { deep: true })

// Re-init on theme change so axis/label colors follow the active palette.
watch(() => theme.theme, () => {
  chart?.dispose()
  chart = null
  render()
})

onBeforeUnmount(() => {
  observer?.disconnect()
  observer = null
  chart?.dispose()
  chart = null
})
</script>

<style scoped>
.admin-echart {
  width: 100%;
  min-height: 180px;
}
</style>
