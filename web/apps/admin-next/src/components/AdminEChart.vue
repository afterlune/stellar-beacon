<template>
  <div ref="container" class="admin-echart" :style="{ height }" />
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import { useThemeStore } from '@/stores/theme'
import { ensureChartTheme } from '@/utils/chart-theme'
import { init, type ECharts } from '@/utils/echarts'

const props = withDefaults(defineProps<{ option: Record<string, unknown>; height?: string }>(), {
  height: '300px'
})

const theme = useThemeStore()
const container = ref<HTMLElement | null>(null)

let chart: ECharts | null = null
let observer: ResizeObserver | null = null

/**
 * ECharts logs a console error when initialized into a zero-sized container,
 * which would break the "console must stay clean" test contract. Defer the
 * first init until the element actually has a box.
 */
function ensureChart(): ECharts | null {
  if (chart) return chart
  const element = container.value
  if (!element || element.clientWidth === 0 || element.clientHeight === 0) return null
  // 用本站设计令牌注册的主题，而不是 ECharts 内置的那套高饱和彩虹色。
  chart = init(element, ensureChartTheme(theme.theme))
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
