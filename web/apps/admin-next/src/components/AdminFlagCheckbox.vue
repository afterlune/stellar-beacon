<template>
  <a-checkbox
    :model-value="modelValue === 1"
    :disabled="disabled"
    @change="(value: boolean | (string | number | boolean)[]) => emit('update:modelValue', value ? 1 : 0)">
    <slot />
  </a-checkbox>
</template>

<script setup lang="ts">
/**
 * Arco's checkbox models a boolean (or an array of values), while the backend
 * contract uses `0 | 1` flags. This adapter keeps views free of casts and
 * guarantees the emitted payload type.
 */
withDefaults(defineProps<{
  modelValue?: number
  disabled?: boolean
}>(), {
  modelValue: 0,
  disabled: false
})

const emit = defineEmits<{ 'update:modelValue': [value: number] }>()
</script>
