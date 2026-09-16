<template>
  <canvas ref="canvas" class="ambient-grid" aria-hidden="true" />
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const canvas = ref<HTMLCanvasElement | null>(null)
let frame = 0
let resizeFrame = 0
let stopped = false
let reducedMotion = false
let lastPaint = 0

type NodePoint = { x: number; y: number; phase: number; radius: number }
let points: NodePoint[] = []

const cssValue = (name: string, fallback: string) => {
  const value = canvas.value ? getComputedStyle(canvas.value).getPropertyValue(name).trim() : ''
  return value || fallback
}

const buildPoints = (width: number, height: number) => {
  const columns = Math.max(7, Math.ceil(width / 180))
  const rows = Math.max(5, Math.ceil(height / 150))
  points = []
  for (let row = 0; row <= rows; row += 1) {
    for (let column = 0; column <= columns; column += 1) {
      const seed = Math.sin(row * 91.7 + column * 37.2) * 43758.5453
      const random = seed - Math.floor(seed)
      points.push({
        x: (column / columns) * width + (random - 0.5) * 72,
        y: (row / rows) * height + (Math.cos(column * 4.1 + row) - 0.5) * 62,
        phase: random * Math.PI * 2,
        radius: random > 0.82 ? 2 : 1
      })
    }
  }
}

const draw = (time: number) => {
  const element = canvas.value
  if (!element) return
  const context = element.getContext('2d')
  if (!context) return
  const width = window.innerWidth
  const height = window.innerHeight
  const grid = cssValue('--fx-grid', 'rgba(170, 255, 90, .08)')
  const node = cssValue('--fx-node', 'rgba(170, 255, 90, .55)')

  context.clearRect(0, 0, width, height)
  context.lineWidth = 1
  context.strokeStyle = grid
  const size = 64
  for (let x = size; x < width; x += size) {
    context.beginPath()
    context.moveTo(x, 0)
    context.lineTo(x, height)
    context.stroke()
  }
  for (let y = size; y < height; y += size) {
    context.beginPath()
    context.moveTo(0, y)
    context.lineTo(width, y)
    context.stroke()
  }

  context.strokeStyle = cssValue('--fx-link', 'rgba(111, 168, 255, .12)')
  points.forEach((point, index) => {
    const next = points[index + 1]
    if (next && index % 3 === 0 && Math.abs(next.y - point.y) < 170) {
      context.beginPath()
      context.moveTo(point.x, point.y)
      context.lineTo(next.x, next.y)
      context.stroke()
    }
  })

  points.forEach((point) => {
    const pulse = reducedMotion ? 0.65 : 0.48 + Math.sin(time * 0.001 + point.phase) * 0.18
    context.globalAlpha = Math.max(0.18, pulse)
    context.fillStyle = node
    context.beginPath()
    context.arc(point.x, point.y, point.radius, 0, Math.PI * 2)
    context.fill()
  })
  context.globalAlpha = 1
}

const paint = (time = 0) => {
  if (stopped) return
  if (!reducedMotion && time - lastPaint < 34) {
    frame = requestAnimationFrame(paint)
    return
  }
  lastPaint = time
  draw(time)
  if (!reducedMotion) frame = requestAnimationFrame(paint)
}

const resize = () => {
  if (resizeFrame) cancelAnimationFrame(resizeFrame)
  resizeFrame = requestAnimationFrame(() => {
    const element = canvas.value
    if (!element) return
    const ratio = Math.min(window.devicePixelRatio || 1, 1.5)
    const width = window.innerWidth
    const height = window.innerHeight
    element.width = Math.floor(width * ratio)
    element.height = Math.floor(height * ratio)
    element.style.width = `${width}px`
    element.style.height = `${height}px`
    element.getContext('2d')?.setTransform(ratio, 0, 0, ratio, 0, 0)
    buildPoints(width, height)
    draw(lastPaint)
  })
}

const handleVisibility = () => {
  if (document.hidden) {
    cancelAnimationFrame(frame)
  } else if (!reducedMotion) {
    frame = requestAnimationFrame(paint)
  }
}

onMounted(() => {
  reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  resize()
  window.addEventListener('resize', resize, { passive: true })
  document.addEventListener('visibilitychange', handleVisibility)
  if (!reducedMotion) frame = requestAnimationFrame(paint)
})

onBeforeUnmount(() => {
  stopped = true
  cancelAnimationFrame(frame)
  cancelAnimationFrame(resizeFrame)
  window.removeEventListener('resize', resize)
  document.removeEventListener('visibilitychange', handleVisibility)
})
</script>

<style scoped>
.ambient-grid {
  position: fixed;
  inset: 0;
  z-index: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  opacity: 0.72;
  mask-image: linear-gradient(180deg, #000 0%, rgba(0, 0, 0, 0.72) 42%, transparent 100%);
}
</style>
