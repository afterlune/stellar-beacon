<template>
  <div
    @click.stop.prevent="handleClick"
    :class="{ 'is-disabled': disabled }"
    :aria-disabled="disabled || undefined"
    class="block cursor-pointer hover:bg-ob-trans my-1 px-4 py-1 font-medium hover:text-ob-bright">
    <slot />
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue'
import { useDropdownStore } from '@/stores/dropdown'

export default defineComponent({
  name: 'DropdownItem',
  props: {
    name: String,
    disabled: Boolean
  },
  setup(props) {
    const dropdownStore = useDropdownStore()
    const handleClick = () => {
      if (props.disabled) return
      dropdownStore.setCommand(String(props.name))
    }
    return { handleClick }
  }
})
</script>

<style scoped>
.is-disabled { cursor: not-allowed; opacity: .42; }
.is-disabled:hover { background: transparent !important; color: inherit !important; }
</style>
