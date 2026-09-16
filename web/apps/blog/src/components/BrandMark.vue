<template>
  <svg
    class="brand-mark"
    viewBox="0 0 48 48"
    :width="size"
    :height="size"
    role="img"
    :aria-label="label">
    <title>{{ label }}</title>
    <rect x="1" y="1" width="46" height="46" rx="3" class="mark-frame" />
    <path d="M9 32.5 19.5 22 27 29.5 39 12" class="mark-path" />
    <circle cx="9" cy="32.5" r="3" class="mark-node mark-node--muted" />
    <circle cx="19.5" cy="22" r="3" class="mark-node" />
    <circle cx="27" cy="29.5" r="3" class="mark-node" />
    <circle cx="39" cy="12" r="4" class="mark-node mark-node--signal" />
    <path d="M9 39h30" class="mark-baseline" />
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ size?: number; label?: string }>(), {
  size: 38,
  label: 'Stellar Beacon'
})
</script>

<style lang="scss" scoped>
.brand-mark {
  display: block;
  flex: 0 0 auto;
}

.mark-frame {
  fill: var(--background-secondary-alt);
  stroke: var(--accent);
  stroke-width: 1;
}

.mark-path {
  fill: none;
  stroke: var(--accent-2);
  stroke-width: 1.5;
}

.mark-node {
  fill: var(--accent);
}

.mark-node--muted {
  fill: var(--text-dim);
}

.mark-node--signal {
  fill: var(--signal);
  stroke: var(--background-secondary-alt);
  stroke-width: 1.5;
}

.mark-baseline {
  stroke: var(--border-strong);
  stroke-width: 1;
}
</style>
