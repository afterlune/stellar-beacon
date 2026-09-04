<template>
  <section
    v-if="visible"
    class="agent-galaxy"
    :data-performance-mode="performanceLimited ? 'limited' : 'full'"
    :data-point-cap="maxPoints"
    :data-measured-fps="measuredFps > 0 ? measuredFps.toFixed(1) : undefined"
    aria-label="Benetnasch 内容星河">
    <div class="agent-galaxy__heading">
      <div>
        <span class="agent-galaxy__eyebrow">CONTENT CONSTELLATION</span>
        <h2>内容星河</h2>
      </div>
      <div class="agent-galaxy__controls">
        <label>
          <span class="sr-only">筛选生命阶段</span>
          <select v-model="stage" :disabled="loading" aria-label="筛选生命阶段">
            <option value="">全部阶段</option>
            <option value="newborn">新生</option>
            <option value="growing">生长</option>
            <option value="settled">沉淀</option>
            <option value="forgotten">遗忘</option>
          </select>
        </label>
        <button type="button" :disabled="loading" @click="refresh">刷新</button>
      </div>
    </div>
    <p v-if="errorMessage" class="agent-galaxy__hint">{{ errorMessage }}</p>
    <p v-else-if="loading && points.length === 0" class="agent-galaxy__hint">正在展开星河……</p>
    <div v-else class="agent-galaxy__canvas-wrap">
      <canvas
        ref="canvas"
        class="agent-galaxy__canvas"
        role="img"
        aria-label="文章二维星河，可拖拽和缩放"
        @pointerdown="pointerDown"
        @pointermove="pointerMove"
        @pointerup="pointerUp"
        @pointercancel="pointerUp"
        @wheel.prevent="wheel"
      />
      <p v-if="points.length === 0" class="agent-galaxy__empty">暂时还没有完成投影的公开文章。</p>
      <span class="agent-galaxy__hint agent-galaxy__hint--overlay">滚轮缩放 · 拖拽浏览</span>
    </div>
    <div class="agent-galaxy__footer">
      <span>{{ points.length }} 个坐标</span>
      <span v-if="performanceLimited" class="agent-galaxy__performance-hint">低性能渲染已启用</span>
      <button v-if="hasMore" type="button" :disabled="loading" @click="loadNextPage">加载更多</button>
      <span v-if="loading && points.length > 0">同步中……</span>
    </div>
  </section>
</template>

<script lang="ts">
import { computed, defineComponent, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import api from '@/api/api'

interface GalaxyPoint {
  articleId: number
  lifeStage: string
  x: number
  y: number
  updatedAt: string
}

interface GalaxyPage {
  records?: GalaxyPoint[]
  hasMore?: boolean
  nextSince?: string
}

const PAGE_SIZE = 100
const POLL_INTERVAL = 60_000

export default defineComponent({
  name: 'AgentGalaxy',
  setup() {
    const canvas = ref<HTMLCanvasElement | null>(null)
    const points = ref<GalaxyPoint[]>([])
    const stage = ref('')
    const loading = ref(false)
    const visible = ref(true)
    const errorMessage = ref('')
    const hasMore = ref(false)
    const page = ref(1)
    const highWaterMark = ref('')
    const performanceLimited = ref(false)
    const measuredFps = ref(0)
    const viewport = { zoom: 1, offsetX: 0, offsetY: 0 }
    const drag = { active: false, x: 0, y: 0 }
    let resizeObserver: ResizeObserver | null = null
    let pollTimer = 0
    let drawFrame = 0
    let fpsFrame = 0
    let previousFrameAt = 0
    let fpsTotal = 0
    let fpsSamples = 0
    let reducedMotion = false

    const maxPoints = computed(() => (performanceLimited.value ? 400 : 2000))

    const draw = () => {
      const element = canvas.value
      if (!element) return
      const context = element.getContext('2d')
      if (!context) return
      const rect = element.getBoundingClientRect()
      if (rect.width <= 0 || rect.height <= 0) return
      const ratio = Math.min(window.devicePixelRatio || 1, 2)
      const width = Math.floor(rect.width * ratio)
      const height = Math.floor(rect.height * ratio)
      if (element.width !== width || element.height !== height) {
        element.width = width
        element.height = height
      }
      context.setTransform(ratio, 0, 0, ratio, 0, 0)
      context.clearRect(0, 0, rect.width, rect.height)
      context.fillStyle = 'rgba(12, 18, 38, 0.28)'
      context.fillRect(0, 0, rect.width, rect.height)
      const visiblePoints = points.value.slice(0, maxPoints.value)
      if (visiblePoints.length === 0) return
      const bounds = visiblePoints.reduce(
        (result, point) => ({
          minX: Math.min(result.minX, point.x),
          maxX: Math.max(result.maxX, point.x),
          minY: Math.min(result.minY, point.y),
          maxY: Math.max(result.maxY, point.y)
        }),
        { minX: Infinity, maxX: -Infinity, minY: Infinity, maxY: -Infinity }
      )
      const rangeX = Math.max(bounds.maxX - bounds.minX, 1)
      const rangeY = Math.max(bounds.maxY - bounds.minY, 1)
      const padding = 28
      const scale = Math.min((rect.width - padding * 2) / rangeX, (rect.height - padding * 2) / rangeY)
      const centerX = rect.width / 2 + viewport.offsetX
      const centerY = rect.height / 2 + viewport.offsetY
      const color: Record<string, string> = {
        newborn: '#7dd3fc',
        growing: '#a7f3d0',
        settled: '#c4b5fd',
        forgotten: '#f9a8d4'
      }
      context.globalCompositeOperation = performanceLimited.value ? 'source-over' : 'lighter'
      for (const point of visiblePoints) {
        const x = centerX + (point.x - (bounds.minX + bounds.maxX) / 2) * scale * viewport.zoom
        const y = centerY + (point.y - (bounds.minY + bounds.maxY) / 2) * scale * viewport.zoom
        if (x < -10 || y < -10 || x > rect.width + 10 || y > rect.height + 10) continue
        context.beginPath()
        context.fillStyle = color[point.lifeStage] || '#bae6fd'
        context.arc(x, y, performanceLimited.value ? 2 : 2.8, 0, Math.PI * 2)
        context.fill()
      }
      context.globalCompositeOperation = 'source-over'
    }

    const scheduleDraw = () => {
      if (drawFrame !== 0) return
      drawFrame = window.requestAnimationFrame(() => {
        drawFrame = 0
        draw()
      })
    }

    const sampleFrameRate = (timestamp: number) => {
      if (document.visibilityState !== 'visible' || performanceLimited.value || reducedMotion || points.value.length === 0) {
        fpsFrame = 0
        return
      }
      // Measure the work users actually pay for. Sampling only the
      // requestAnimationFrame cadence can report a healthy FPS even when a
      // dense Canvas draw is already consuming the frame budget.
      draw()
      if (previousFrameAt > 0) {
        const elapsed = timestamp - previousFrameAt
        if (elapsed > 0 && elapsed < 250) {
          fpsTotal += 1000 / elapsed
          fpsSamples += 1
        }
      }
      previousFrameAt = timestamp
      if (fpsSamples >= 20) {
        const averageFps = fpsTotal / fpsSamples
        measuredFps.value = Math.round(averageFps * 10) / 10
        if (averageFps < 45) {
          performanceLimited.value = true
          scheduleDraw()
        }
        fpsFrame = 0
        return
      }
      fpsFrame = window.requestAnimationFrame(sampleFrameRate)
    }

    const startFrameRateSampling = () => {
      if (fpsFrame !== 0 || performanceLimited.value || reducedMotion || document.visibilityState !== 'visible' || points.value.length === 0) return
      previousFrameAt = 0
      fpsTotal = 0
      fpsSamples = 0
      fpsFrame = window.requestAnimationFrame(sampleFrameRate)
    }

    const stopFrameRateSampling = () => {
      if (fpsFrame !== 0) window.cancelAnimationFrame(fpsFrame)
      fpsFrame = 0
      previousFrameAt = 0
      fpsTotal = 0
      fpsSamples = 0
    }

    const mergePoints = (incoming: GalaxyPoint[], replace: boolean) => {
      const merged = replace ? new Map<number, GalaxyPoint>() : new Map(points.value.map((point) => [point.articleId, point]))
      incoming.forEach((point) => {
        if (Number.isFinite(point.x) && Number.isFinite(point.y) && point.articleId > 0) merged.set(point.articleId, point)
      })
      points.value = Array.from(merged.values()).sort((left, right) => left.articleId - right.articleId)
    }

    const requestPage = async (current: number, since = '', replace = false) => {
      loading.value = true
      errorMessage.value = ''
      try {
        const { data } = await api.getAgentGalaxy({
          current,
          size: PAGE_SIZE,
          stage: stage.value || undefined,
          since: since || undefined
        })
        if (!data?.flag) {
          visible.value = false
          return
        }
        const result = (data.data || {}) as GalaxyPage
        mergePoints(Array.isArray(result.records) ? result.records : [], replace)
        hasMore.value = result.hasMore === true
        page.value = current
        if (result.nextSince && (!highWaterMark.value || result.nextSince > highWaterMark.value)) highWaterMark.value = result.nextSince
        await nextTick()
        scheduleDraw()
        startFrameRateSampling()
      } catch (_error) {
        errorMessage.value = '星河暂时离线，请稍后再试。'
      } finally {
        loading.value = false
      }
    }

    const refresh = () => {
      page.value = 1
      highWaterMark.value = ''
      viewport.zoom = 1
      viewport.offsetX = 0
      viewport.offsetY = 0
      void requestPage(1, '', true)
    }

    const loadNextPage = () => {
      if (loading.value || !hasMore.value) return
      void requestPage(page.value + 1)
    }

    const poll = () => {
      if (document.visibilityState !== 'visible' || loading.value || !highWaterMark.value) return
      void requestPage(1, highWaterMark.value)
    }

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'hidden') {
        stopFrameRateSampling()
        return
      }
      scheduleDraw()
      startFrameRateSampling()
    }

    const pointerDown = (event: PointerEvent) => {
      if (!canvas.value) return
      drag.active = true
      drag.x = event.clientX
      drag.y = event.clientY
      canvas.value.setPointerCapture(event.pointerId)
    }

    const pointerMove = (event: PointerEvent) => {
      if (!drag.active) return
      viewport.offsetX += event.clientX - drag.x
      viewport.offsetY += event.clientY - drag.y
      drag.x = event.clientX
      drag.y = event.clientY
      scheduleDraw()
    }

    const pointerUp = () => {
      drag.active = false
    }

    const wheel = (event: WheelEvent) => {
      const direction = event.deltaY < 0 ? 1.12 : 0.89
      viewport.zoom = Math.max(0.5, Math.min(4, viewport.zoom * direction))
      scheduleDraw()
    }

    watch(stage, refresh)
    onMounted(() => {
      const memory = Number((navigator as Navigator & { deviceMemory?: number }).deviceMemory || 8)
      reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
      performanceLimited.value = reducedMotion || memory <= 2 || navigator.hardwareConcurrency <= 2
      resizeObserver = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(scheduleDraw) : null
      if (canvas.value && resizeObserver) resizeObserver.observe(canvas.value)
      window.addEventListener('resize', scheduleDraw)
      document.addEventListener('visibilitychange', handleVisibilityChange)
      pollTimer = window.setInterval(poll, POLL_INTERVAL)
      refresh()
    })
    onBeforeUnmount(() => {
      resizeObserver?.disconnect()
      window.removeEventListener('resize', scheduleDraw)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      window.clearInterval(pollTimer)
      if (drawFrame !== 0) window.cancelAnimationFrame(drawFrame)
      stopFrameRateSampling()
    })

    return {
      canvas,
      points,
      stage,
      loading,
      visible,
      errorMessage,
      hasMore,
      maxPoints,
      performanceLimited,
      measuredFps,
      refresh,
      loadNextPage,
      pointerDown,
      pointerMove,
      pointerUp,
      wheel
    }
  }
})
</script>

<style lang="scss" scoped>
.agent-galaxy {
  margin-bottom: 2rem;
  padding: 1rem 1.25rem;
  border: 1px solid var(--bg-accent-55);
  border-radius: 1rem;
  background: var(--background-secondary);
  box-shadow: var(--accent-shadow);
}

.agent-galaxy__heading,
.agent-galaxy__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.agent-galaxy__eyebrow {
  color: var(--text-accent);
  font-size: 0.68rem;
  letter-spacing: 0.12em;
}

.agent-galaxy h2 {
  margin: 0.2rem 0 0;
  color: var(--text-bright);
  font-size: 1.25rem;
}

.agent-galaxy__controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.agent-galaxy select,
.agent-galaxy button {
  border: 1px solid var(--bg-accent-55);
  border-radius: 0.5rem;
  padding: 0.35rem 0.55rem;
  background: var(--background-primary);
  color: var(--text-normal);
  cursor: pointer;
}

.agent-galaxy button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.agent-galaxy__canvas-wrap {
  position: relative;
  height: min(52vw, 420px);
  min-height: 240px;
  margin-top: 0.85rem;
  overflow: hidden;
  border-radius: 0.75rem;
  background: radial-gradient(circle at center, rgba(70, 80, 160, 0.25), rgba(8, 12, 28, 0.8));
}

.agent-galaxy__canvas {
  display: block;
  width: 100%;
  height: 100%;
  touch-action: none;
  cursor: grab;
}

.agent-galaxy__canvas:active {
  cursor: grabbing;
}

.agent-galaxy__hint,
.agent-galaxy__empty,
.agent-galaxy__footer,
.agent-galaxy__performance-hint {
  color: var(--text-normal);
  font-size: 0.8rem;
}

.agent-galaxy__performance-hint {
  color: var(--text-accent);
}

.agent-galaxy__hint--overlay {
  position: absolute;
  right: 0.65rem;
  bottom: 0.55rem;
  margin: 0;
  opacity: 0.7;
}

.agent-galaxy__empty {
  position: absolute;
  top: 50%;
  left: 50%;
  margin: 0;
  transform: translate(-50%, -50%);
}

.agent-galaxy__footer {
  margin-top: 0.65rem;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>
