<template>
  <svg
    class="brand-mark"
    viewBox="0 0 100 100"
    :width="size"
    :height="size"
    role="img"
    :aria-label="label">
    <title>{{ label }}</title>
    <defs>
      <linearGradient :id="`${uid}-sky`" x1="0" y1="0" x2=".6" y2="1">
        <stop offset="0" stop-color="#1a1d3a" />
        <stop offset="1" stop-color="#080a14" />
      </linearGradient>
      <radialGradient :id="`${uid}-cyan`">
        <stop offset="0" stop-color="#22d3ee" stop-opacity=".9" />
        <stop offset="1" stop-color="#22d3ee" stop-opacity="0" />
      </radialGradient>
      <radialGradient :id="`${uid}-violet`">
        <stop offset="0" stop-color="#6d4bff" stop-opacity=".85" />
        <stop offset="1" stop-color="#6d4bff" stop-opacity="0" />
      </radialGradient>
      <radialGradient :id="`${uid}-magenta`">
        <stop offset="0" stop-color="#ff2d95" stop-opacity="1" />
        <stop offset="1" stop-color="#ff2d95" stop-opacity="0" />
      </radialGradient>
      <radialGradient :id="`${uid}-pocket`">
        <stop offset="0" stop-color="#05060d" stop-opacity=".5" />
        <stop offset="1" stop-color="#05060d" stop-opacity="0" />
      </radialGradient>
      <linearGradient :id="`${uid}-trail`" x1="0" y1="1" x2="1" y2="0">
        <stop offset="0" stop-color="#22d3ee" />
        <stop offset=".5" stop-color="#a78bfa" />
        <stop offset="1" stop-color="#ff2d95" />
      </linearGradient>
      <linearGradient :id="`${uid}-star`" x1=".15" y1="0" x2=".85" y2="1">
        <stop offset="0" stop-color="#ffffff" />
        <stop offset="1" stop-color="#bdf3ff" />
      </linearGradient>
    </defs>

    <rect width="100" height="100" rx="23.4" :fill="`url(#${uid}-sky)`" />
    <circle cx="12" cy="100" r="47" :fill="`url(#${uid}-cyan)`" />
    <circle cx="50" cy="103" r="41" :fill="`url(#${uid}-violet)`" />
    <circle cx="91" cy="100" r="43" :fill="`url(#${uid}-magenta)`" />

    <path
      d="M19 86 Q44 68 73 20"
      fill="none"
      :stroke="`url(#${uid}-trail)`"
      stroke-width="3"
      stroke-linecap="round"
      opacity=".42" />

    <circle cx="33" cy="64" r="11" :fill="`url(#${uid}-pocket)`" />
    <circle cx="33" cy="64" r="9.5" :fill="`url(#${uid}-cyan)`" />
    <circle cx="33" cy="64" r="4" fill="#e6faff" />

    <circle cx="50" cy="46" r="11" :fill="`url(#${uid}-pocket)`" />
    <circle cx="50" cy="46" r="9.5" :fill="`url(#${uid}-cyan)`" />
    <circle cx="50" cy="46" r="5" fill="#ffffff" />

    <circle cx="70" cy="26" r="25" :fill="`url(#${uid}-magenta)`" />
    <path
      d="M70 8.5 C72 20 77 25 88.5 27 C77 29 72 34 70 45.5 C68 34 63 29 51.5 27 C63 25 68 20 70 8.5 Z"
      :fill="`url(#${uid}-star)`" />
  </svg>
</template>

<script setup lang="ts">
import { useId } from 'vue'

withDefaults(defineProps<{ size?: number; label?: string }>(), {
  size: 38,
  label: '星际信标 · Stellar Beacon'
})

// 每个实例一套渐变 ID，同页多枚图标互不覆盖。
const uid = useId()
</script>

<style lang="scss" scoped>
.brand-mark {
  display: block;
  flex: 0 0 auto;
  border-radius: 23.4%;
  /* 深色底 + 内高光，让这枚标在浅色和深色页面上都能立住。 */
  box-shadow: 0 6px 18px -8px rgba(109, 75, 255, 0.62), inset 0 1px 0 rgba(255, 255, 255, 0.14);
}
</style>
