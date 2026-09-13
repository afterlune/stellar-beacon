<template>
  <div class="admin-page-header">
    <div class="admin-page-header-copy">
      <div v-if="eyebrow" class="admin-page-eyebrow">{{ eyebrow }}</div>
      <h2>{{ title }}</h2>
      <p v-if="description">{{ description }}</p>
    </div>
    <div v-if="$slots.actions || $slots.default" class="admin-page-actions">
      <a-space wrap>
        <slot name="actions" />
        <slot />
      </a-space>
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  title: string
  description?: string
  eyebrow?: string
}>(), {
  description: '',
  eyebrow: 'STELLAR BEACON / 管理台'
})
</script>
