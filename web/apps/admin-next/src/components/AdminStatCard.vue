<template>
  <a-card class="admin-card admin-stat-card" :bordered="false">
    <span class="admin-stat-icon" :class="`admin-tone-${tone}`" aria-hidden="true">
      <component :is="icon" />
    </span>
    <span class="admin-stat-copy">
      <span class="admin-stat-label">{{ label }}</span>
      <strong class="admin-stat-value">{{ value }}</strong>
      <span v-if="caption" class="admin-stat-caption" :title="caption">{{ caption }}</span>
    </span>
  </a-card>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

type Tone = 'blue' | 'green' | 'warm' | 'violet' | 'info' | 'danger'

withDefaults(defineProps<{
  label: string
  value: string | number
  caption?: string
  icon: Component
  tone?: Tone
}>(), {
  caption: '',
  tone: 'blue'
})
</script>
