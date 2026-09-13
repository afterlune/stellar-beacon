<template>
  <p :class="rootClass">
    <svg-icon v-if="icon && side === 'left'" :icon-class="icon" class="inline-block mr-2" />
    <span :class="titleClass">{{ t(titleStr) }}</span>
    <svg-icon v-if="icon && side === 'right'" :icon-class="icon" class="inline-block ml-2" />
    <span :class="lineClass" />
  </p>
</template>

<script lang="ts">
import { computed, defineComponent, toRefs } from 'vue'
import { useI18n } from 'vue-i18n'

export default defineComponent({
  name: 'SubTitle',
  props: {
    title: {
      type: String,
      default: '',
      requried: true
    },
    side: {
      type: String,
      default: 'left'
    },
    icon: String,
    // Sidebar headings are a quiet label; section headings in the article body
    // keep the larger standalone treatment.
    compact: {
      type: Boolean,
      default: false
    }
  },
  setup(props) {
    const { t } = useI18n()
    const titleStr = toRefs(props).title
    const side = toRefs(props).side
    const compact = toRefs(props).compact

    return {
      rootClass: computed(() => {
        if (compact.value) {
          return { relative: true, flex: true, 'items-center': true, 'sub-title-compact': true }
        }
        return {
          relative: true,
          flex: true,
          'items-center': true,
          'pb-2': true,
          'mb-4': true,
          'text-xl': true,
          'text-ob-bright': true,
          uppercase: true
        }
      }),
      titleClass: computed(() => {
        return {
          'w-full': true,
          block: true,
          'text-right': side.value === 'right' ? true : false
        }
      }),
      lineClass: computed(() => {
        return {
          absolute: true,
          'bottom-0': true,
          'h-1': true,
          'w-14': true,
          'rounded-full': true,
          'brand-rule': true,
          'right-0': side.value === 'right' ? true : false
        }
      }),
      titleStr,
      t
    }
  }
})
</script>

<style lang="scss" scoped>
.sub-title-compact {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-dim);
  padding-bottom: 10px;
  margin-bottom: 14px;
  border-bottom: 1px solid var(--border-hairline);

  /* Use the border as the sidebar heading separator. */
  .brand-rule {
    display: none;
  }
}
</style>
