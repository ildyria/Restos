<script setup lang="ts">
// Read-only star display with half-star precision.
const props = withDefaults(defineProps<{ value: number | null | undefined, size?: number }>(), { size: 16 })
// Averages like 4.375 are drawn to the nearest half star.
const pct = computed(() => Math.max(0, Math.min(5, Math.round((props.value ?? 0) * 2) / 2)) / 5 * 100)
const path = 'M12 2.5l2.94 5.96 6.58.96-4.76 4.64 1.12 6.55L12 17.52l-5.88 3.09 1.12-6.55L2.48 9.42l6.58-.96L12 2.5z'
</script>

<template>
  <span
    class="relative inline-flex align-middle"
    role="img"
    :aria-label="value == null ? 'Not rated' : `${value} out of 5`"
  >
    <span class="flex text-neutral-300 dark:text-neutral-700">
      <svg v-for="i in 5" :key="i" :width="size" :height="size" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path :d="path" /></svg>
    </span>
    <span class="absolute inset-y-0 left-0 flex overflow-hidden text-primary" :style="{ width: `${pct}%` }">
      <svg v-for="i in 5" :key="i" class="shrink-0" :width="size" :height="size" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path :d="path" /></svg>
    </span>
  </span>
</template>
