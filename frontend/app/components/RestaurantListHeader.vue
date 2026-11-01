<script setup lang="ts">
import { RATING_CRITERIA } from '~~/shared/types'
</script>

<template>
  <!-- Column labels for the wide RestaurantRow layout; the columns must match it. -->
  <div class="hidden grid-cols-[2.75rem_6rem_minmax(0,1fr)_20.5rem_3.5rem] gap-5 border-b border-default px-3 pb-2 text-[11px] uppercase tracking-normal text-dimmed md:grid">
    <span /><span /><span />
    <div class="grid grid-cols-5 gap-2"><span v-for="c in RATING_CRITERIA" :key="c.key">{{ c.label }}</span></div>
    <span class="text-right">Score</span>
  </div>
</template>
