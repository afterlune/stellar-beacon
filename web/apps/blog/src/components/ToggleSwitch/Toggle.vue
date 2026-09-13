<template>
  <div class="toggler" @click="changeStatus">
    <div class="toggle-track"></div>
    <div class="slider" :class="{ 'slider-active': toggleStyle.active }" :style="{ transform: toggleStyle.transform }">
      <slot />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, reactive, toRefs } from 'vue'

export default defineComponent({
  name: 'ObToggle',
  props: ['status'],
  emits: ['changeStatus'],
  setup(props, { emit }) {
    let { status } = toRefs(props)
    onMounted(() => {
      changeTransform()
    })
    // Use the active theme's brand color for the on state.
    let toggleStyle = reactive({
      transform: '',
      active: false
    })
    let toggleStatus = status.value
    const changeStatus = () => {
      toggleStatus = !toggleStatus
      changeTransform()
      emit('changeStatus', toggleStatus)
    }
    const changeTransform = () => {
      toggleStyle.transform = `translateX(${toggleStatus ? '18px' : '0'})`
      toggleStyle.active = Boolean(toggleStatus)
    }
    return {
      toggleStyle,
      changeStatus
    }
  }
})
</script>

<style lang="scss" scoped>
.toggler {
  @apply relative;
  width: 40px;
  height: 22px;
  background-color: var(--surface-hover);
  border-radius: 24px;
  border: 3px solid var(--border-hairline);
  box-sizing: border-box;
  transition: background-color 250ms ease;
}
.slider {
  top: -6px;
  left: -6px;
  width: 28px;
  height: 28px;
  background-color: var(--text-faint);
  border-radius: 50%;
  transition: all 250ms cubic-bezier(0.4, 0.03, 0, 1) 0s;
  @apply absolute;
  box-shadow: var(--elev-1);
}
.slider-active {
  background-color: transparent;
  background-image: var(--brand-gradient);
}
</style>
