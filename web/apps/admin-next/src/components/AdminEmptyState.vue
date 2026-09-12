<template>
  <section class="admin-empty-state">
    <span class="admin-empty-state-icon" aria-hidden="true"><component :is="icon" /></span>
    <strong>{{ title }}</strong>
    <p v-if="description">{{ description }}</p>
    <div v-if="$slots.default" class="admin-empty-state-actions"><slot /></div>
  </section>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { IconEmpty } from '@arco-design/web-vue/es/icon'

withDefaults(defineProps<{
  title: string
  description?: string
  icon?: Component
}>(), {
  description: '',
  icon: () => IconEmpty
})
</script>
