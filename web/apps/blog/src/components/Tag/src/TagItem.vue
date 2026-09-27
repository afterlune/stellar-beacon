<template>
  <router-link
    class="tag-pill"
    :class="`tag-pill-${size || 'md'}`"
    :to="{ path: '/tags/' + id, query: { tagName: name } }">
    <em class="tag-pill-hash">#</em>
    <span class="tag-pill-name">{{ name }}</span>
    <span class="tag-pill-count">{{ count }}</span>
  </router-link>
</template>

<script lang="ts">
import { defineComponent } from 'vue'

export default defineComponent({
  name: 'ObTagItem',
  props: ['id', 'name', 'count', 'size']
})
</script>

<style lang="scss" scoped>
/* Keep the tag count secondary to the tag name. */
.tag-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0 8px 8px 0;
  padding: 5px 12px;
  border-radius: var(--radius-pill);
  background-color: var(--surface-2);
  border: none;
  color: var(--text-normal);
  line-height: 1.4;
  transition: background-color 200ms ease, color 200ms ease, box-shadow 200ms ease;
}

.tag-pill:hover {
  opacity: 1;
  background-color: color-mix(in srgb, var(--text-accent) 14%, transparent);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--text-accent) 45%, transparent);
  color: var(--text-bright);
}

.tag-pill-hash {
  font-style: normal;
  color: var(--text-faint);
}

.tag-pill-count {
  font-size: 0.85em;
  color: var(--text-dim);
  font-variant-numeric: tabular-nums;
}

.tag-pill-xs {
  font-size: 0.75rem;
  padding: 3px 10px;
}
.tag-pill-sm {
  font-size: 0.875rem;
}
.tag-pill-md {
  font-size: 1rem;
}
.tag-pill-lg {
  font-size: 1.125rem;
  padding: 6px 14px;
}
.tag-pill-xl {
  font-size: 1.125rem;
  padding: 7px 16px;
}
.tag-pill-2xl {
  font-size: 1.25rem;
  padding: 8px 18px;
}
</style>
