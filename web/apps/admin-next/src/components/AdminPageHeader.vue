<template>
  <div class="admin-page-header">
    <div class="admin-page-header-copy">
      <div v-if="eyebrowText" class="admin-page-eyebrow">{{ eyebrowText }}</div>
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
import { computed } from 'vue'

import { t } from '@/i18n'

const props = withDefaults(defineProps<{
  title: string
  description?: string
  /** 传空字符串可以完全隐藏眉标行。 */
  eyebrow?: string
}>(), {
  description: '',
  eyebrow: ''
})

// 眉标默认是品牌名 + 栏目名：跟随语言，且不写死中文。
const eyebrowText = computed(() => props.eyebrow || `${t('shell.brandTagline')} / ${t('shell.console')}`)
</script>
