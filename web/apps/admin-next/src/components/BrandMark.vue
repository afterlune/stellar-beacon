<template>
  <svg
    class="admin-brand-mark"
    viewBox="0 0 100 100"
    aria-hidden="true"
    focusable="false">
    <defs>
      <linearGradient :id="`${uid}-sky`" x1="0" y1="0" x2=".6" y2="1">
        <stop offset="0" stop-color="#5f74c8" />
        <stop offset=".55" stop-color="#4a5aa8" />
        <stop offset="1" stop-color="#3a4684" />
      </linearGradient>
      <radialGradient :id="`${uid}-cyan`">
        <stop offset="0" stop-color="#7ff0ff" stop-opacity=".85" />
        <stop offset="1" stop-color="#7ff0ff" stop-opacity="0" />
      </radialGradient>
      <radialGradient :id="`${uid}-violet`">
        <stop offset="0" stop-color="#a78bfa" stop-opacity=".7" />
        <stop offset="1" stop-color="#a78bfa" stop-opacity="0" />
      </radialGradient>
      <radialGradient :id="`${uid}-magenta`">
        <stop offset="0" stop-color="#ff6fbe" stop-opacity=".95" />
        <stop offset="1" stop-color="#ff6fbe" stop-opacity="0" />
      </radialGradient>
      <linearGradient :id="`${uid}-trail`" x1="0" y1="1" x2="1" y2="0">
        <stop offset="0" stop-color="#63e3f5" />
        <stop offset=".5" stop-color="#c4b5fd" />
        <stop offset="1" stop-color="#ff9ad0" />
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
      stroke-width="3.4"
      stroke-linecap="round"
      opacity=".62" />

    <circle cx="33" cy="64" r="5.4" fill="#ffffff" opacity=".92" />
    <circle cx="50" cy="46" r="6.4" fill="#ffffff" />

    <circle cx="70" cy="26" r="23" :fill="`url(#${uid}-magenta)`" />
    <path
      d="M70 8.5 C72 20 77 25 88.5 27 C77 29 72 34 70 45.5 C68 34 63 29 51.5 27 C63 25 68 20 70 8.5 Z"
      fill="#ffffff" />
  </svg>
</template>

<script setup lang="ts">
import { useId } from 'vue'

// 尺寸由 .admin-brand-mark（styles.css）决定：侧栏 32px、登录页 36px。
const uid = useId()
</script>
