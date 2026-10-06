<script setup lang="ts">
import { RATING_CRITERIA } from '~~/shared/types'
import type { Restaurant } from '~~/shared/types'

defineProps<{ restaurant: Restaurant, compact?: boolean }>()
</script>

<template>
  <dl class="grid gap-x-6 gap-y-2" :class="compact ? 'grid-cols-2' : 'grid-cols-1'">
    <div v-for="c in RATING_CRITERIA" :key="c.key" class="flex items-center gap-3">
      <dt class="w-16 shrink-0 text-xs uppercase tracking-wide text-muted">{{ c.label }}</dt>
      <dd class="flex flex-1 items-center gap-2">
        <span class="h-1 flex-1 overflow-hidden rounded-full bg-stone-200 dark:bg-stone-800">
          <span
            class="block h-full rounded-full bg-primary"
            :style="{ width: `${((restaurant[c.key] ?? 0) / 5) * 100}%` }"
          />
        </span>
        <span class="w-7 text-right text-sm tabular-nums">{{ formatScore(restaurant[c.key]) }}</span>
      </dd>
    </div>
  </dl>
</template>
