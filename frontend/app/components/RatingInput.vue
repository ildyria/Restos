<script setup lang="ts">
// Click the left half of a star for x.5, the right half for a whole point.
const model = defineModel<number | null>({ default: null })
defineProps<{ label?: string }>()

const hover = ref<number | null>(null)
const shown = computed(() => hover.value ?? model.value ?? 0)
const path = 'M12 2.5l2.94 5.96 6.58.96-4.76 4.64 1.12 6.55L12 17.52l-5.88 3.09 1.12-6.55L2.48 9.42l6.58-.96L12 2.5z'

function fillFor(i: number) {
  return Math.max(0, Math.min(1, (shown.value - (i - 1)))) * 100
}

function onKey(e: KeyboardEvent) {
  const cur = model.value ?? 0
  if (e.key === 'ArrowRight' || e.key === 'ArrowUp') {
    model.value = Math.min(5, cur + 0.5)
  } else if (e.key === 'ArrowLeft' || e.key === 'ArrowDown') {
    model.value = cur <= 1 ? null : cur - 0.5
  } else if (e.key === 'Backspace' || e.key === 'Delete') {
    model.value = null
  } else {
    return
  }
  e.preventDefault()
}
</script>

<template>
  <div class="flex items-center gap-3">
    <div
      class="flex rounded outline-none focus-visible:ring-2 focus-visible:ring-primary"
      role="slider"
      tabindex="0"
      :aria-label="label"
      aria-valuemin="1"
      aria-valuemax="5"
      :aria-valuenow="model ?? undefined"
      @keydown="onKey"
      @mouseleave="hover = null"
    >
      <span v-for="i in 5" :key="i" class="relative inline-flex size-7 cursor-pointer">
        <svg class="text-neutral-300 dark:text-neutral-700" width="28" height="28" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path :d="path" /></svg>
        <span class="pointer-events-none absolute inset-y-0 left-0 overflow-hidden text-primary" :style="{ width: `${fillFor(i)}%` }">
          <svg class="max-w-none" width="28" height="28" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path :d="path" /></svg>
        </span>
        <span class="absolute inset-y-0 left-0 w-1/2" @mouseenter="hover = i - 0.5" @click="model = i - 0.5" />
        <span class="absolute inset-y-0 right-0 w-1/2" @mouseenter="hover = i" @click="model = i" />
      </span>
    </div>
    <span class="w-8 text-sm tabular-nums text-muted">{{ model == null ? '–' : model.toFixed(1) }}</span>
    <UButton
      v-if="model != null"
      icon="i-lucide-x"
      size="xs"
      color="neutral"
      variant="ghost"
      aria-label="Clear rating"
      @click="model = null"
    />
  </div>
</template>
