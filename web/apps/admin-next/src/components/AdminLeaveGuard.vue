<template>
  <a-modal
    :visible="visible"
    :title="title"
    :ok-text="okText"
    :cancel-text="cancelText"
    :mask-closable="false"
    :esc-to-close="true"
    width="420px"
    @ok="emit('ok')"
    @cancel="emit('cancel')">
    <slot />
  </a-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import { t } from '@/i18n'

/**
 * 未保存修改的离开确认框。
 *
 * 抽成组件而不是在编辑器里各写一份：两个编辑器的确认文案与按钮语义必须一致，
 * 而且这里的按钮刻意不用「确定」，避免和 E2E 里页面级 `确定` 定位产生歧义。
 */
const props = withDefaults(defineProps<{
  visible: boolean
  title?: string
  okText?: string
  cancelText?: string
}>(), {
  title: '',
  okText: '',
  cancelText: ''
})

// 缺省文案走词条（语言切换后跟着变）；显式传入时以调用方为准。
const title = computed(() => props.title || t('common.unsavedTitle'))
const okText = computed(() => props.okText || t('common.unsavedLeave'))
const cancelText = computed(() => props.cancelText || t('common.unsavedStay'))

const emit = defineEmits<{ ok: []; cancel: [] }>()
</script>
