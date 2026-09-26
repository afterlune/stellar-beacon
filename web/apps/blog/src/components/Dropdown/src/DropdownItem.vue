<template>
  <button
    type="button"
    @click.stop.prevent="handleClick"
    :disabled="disabled"
    class="block w-full cursor-pointer text-left hover:bg-ob-trans my-1 px-4 py-1 font-medium hover:text-ob-bright">
    <slot />
  </button>
</template>

<script lang="ts">
import { defineComponent, inject } from 'vue'
import { useDropdownStore } from '@/stores/dropdown'

export default defineComponent({
  name: 'DropdownItem',
  props: {
    name: String,
    disabled: Boolean
  },
  setup(props) {
    const dropdownStore = useDropdownStore()
    const sharedState = inject<{ active: boolean }>('sharedState')
    const handleClick = () => {
      if (props.disabled) return
      dropdownStore.setCommand(String(props.name))
      if (sharedState) sharedState.active = false
    }
    return { handleClick }
  }
})
</script>

<style scoped>
button:focus-visible {
  outline: 2px solid var(--color-ob);
  outline-offset: -2px;
}
.is-disabled { cursor: not-allowed; opacity: .42; }
.is-disabled:hover { background: transparent !important; color: inherit !important; }
</style>
