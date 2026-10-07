<script setup lang="ts">
import type { RestaurantSummary } from '~~/shared/types'
import { RATING_CRITERIA } from '~~/shared/types'

const props = defineProps<{ restaurant: RestaurantSummary, showRank?: boolean, open?: boolean, highlighted?: boolean }>()
defineEmits<{ toggle: [], hover: [on: boolean] }>()

const r = computed(() => props.restaurant)
const cats = computed(() => RATING_CRITERIA.map(c => {
  const v = r.value[c.key]
  return { key: c.key, label: c.label, text: formatScore(v), pct: `${((v ?? 0) / 5) * 100}%` }
}))
const verdict = computed(() => returnVerdict(r.value.scores))
const verdictClass = computed(() => ({
  all: ['bg-success/10 text-success', 'bg-success'],
  some: ['bg-primary/10 text-primary', 'bg-primary'],
  none: ['bg-elevated text-muted', 'bg-(--ui-text-dimmed)']
})[verdict.value?.tone ?? 'none'])
const meta = computed(() => [
  r.value.city,
  priceLabel(r.value.price_level),
  r.value.last_visit && formatDate(r.value.last_visit, { day: 'numeric', month: 'short', year: 'numeric' })
].filter(Boolean) as string[])
const visits = computed(() => `${r.value.visit_count} ${r.value.visit_count === 1 ? 'visit' : 'visits'}`)
</script>

<template>
  <div
    class="border-b border-default transition-colors"
    :class="highlighted || open ? 'bg-elevated/60' : ''"
    @mouseenter="$emit('hover', true)"
    @mouseleave="$emit('hover', false)"
  >
    <!-- Desktop / landscape tablet -->
    <div
      class="hidden cursor-pointer grid-cols-[2.75rem_6rem_minmax(0,1fr)_20.5rem_3.5rem] items-center gap-5 px-3 py-[18px] md:grid"
      role="button"
      tabindex="0"
      :aria-expanded="open"
      @click="$emit('toggle')"
      @keydown.enter.prevent="$emit('toggle')"
      @keydown.space.prevent="$emit('toggle')"
    >
      <span class="rank-numeral text-right text-4xl text-dimmed">{{ showRank ? r.rank_position : '' }}</span>

      <div class="h-[72px] overflow-hidden rounded-md bg-elevated">
        <img v-if="r.cover_key" :src="mediaUrl(r.cover_key)" :alt="r.name" class="size-full object-cover" loading="lazy">
        <div v-else class="grid size-full place-items-center text-dimmed"><UIcon name="i-lucide-utensils-crossed" class="size-5" /></div>
      </div>

      <div class="min-w-0">
        <h3 class="font-display text-2xl leading-[1.1]">{{ r.name }}<MichelinStars v-if="r.michelin_stars" :count="r.michelin_stars" class="ml-2 text-base" /></h3>
        <p class="mt-1.5 flex flex-wrap gap-x-[7px] text-sm text-muted">
          <template v-for="(m, i) in meta" :key="i"><span v-if="i" class="text-dimmed">·</span><span>{{ m }}</span></template>
        </p>
        <span v-if="verdict" class="mt-2 inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs" :class="verdictClass[0]">
          <span class="size-1.5 rounded-full" :class="verdictClass[1]" />{{ verdict.label }}
        </span>
      </div>

      <div class="grid grid-cols-5 gap-2">
        <div v-for="c in cats" :key="c.key">
          <div class="lining text-base">{{ c.text }}</div>
          <div class="mt-[5px] h-[3px] overflow-hidden rounded-sm bg-accented/60">
            <div class="h-full bg-primary" :style="{ width: c.pct }" />
          </div>
        </div>
      </div>

      <span class="lining text-right text-2xl">{{ formatScore(r.overall) }}</span>
    </div>

    <div v-if="open" class="hidden grid-cols-[2.75rem_6rem_minmax(0,1fr)_20.5rem_3.5rem] gap-5 px-3 pb-5 md:grid">
      <span /><span />
      <div class="text-sm leading-normal text-muted">
        <template v-if="r.address || r.city">{{ [r.address, r.city].filter(Boolean).join(', ') }}<br></template>
        <template v-if="r.visit_count">{{ visits }}</template>
        <NuxtLink :to="`/r/${r.slug}`" class="mt-2 block text-primary hover:underline">View visits &amp; dishes →</NuxtLink>
      </div>
      <div class="flex flex-col gap-2">
        <div v-for="p in r.scores" :key="p.user_id" class="lining grid grid-cols-5 gap-2 text-sm text-toned">
          <span v-for="c in RATING_CRITERIA" :key="c.key">{{ formatScore(p[c.key]) }}</span>
        </div>
      </div>
      <div class="flex flex-col gap-2 text-right text-sm text-muted">
        <span v-for="p in r.scores" :key="p.user_id" class="truncate" :title="p.name">{{ p.name }}</span>
      </div>
    </div>

    <!-- Mobile / portrait tablet -->
    <div
      class="cursor-pointer py-[18px] md:hidden"
      role="button"
      tabindex="0"
      :aria-expanded="open"
      @click="$emit('toggle')"
      @keydown.enter.prevent="$emit('toggle')"
    >
      <div class="grid grid-cols-[1.75rem_minmax(0,1fr)_auto] items-start gap-3">
        <span class="rank-numeral text-2xl text-dimmed">{{ showRank ? r.rank_position : '' }}</span>
        <div class="min-w-0">
          <h3 class="font-display text-2xl leading-[1.1]">{{ r.name }}<MichelinStars v-if="r.michelin_stars" :count="r.michelin_stars" class="ml-2 text-base" /></h3>
          <p class="mt-[5px] text-sm text-muted">{{ meta.join(' · ') }}</p>
        </div>
        <span class="lining text-2xl">{{ formatScore(r.overall) }}</span>
      </div>
      <div class="ml-10 mt-3 h-[150px] overflow-hidden rounded-lg bg-elevated">
        <img v-if="r.cover_key" :src="mediaUrl(r.cover_key)" :alt="r.name" class="size-full object-cover" loading="lazy">
        <div v-else class="grid size-full place-items-center text-dimmed"><UIcon name="i-lucide-utensils-crossed" class="size-6" /></div>
      </div>
      <div class="ml-10 mt-3 grid grid-cols-5 gap-2">
        <div v-for="c in cats" :key="c.key">
          <div class="text-xs text-dimmed">{{ c.label }}</div>
          <div class="lining text-base">{{ c.text }}</div>
          <div class="mt-1 h-[3px] overflow-hidden rounded-sm bg-accented/60"><div class="h-full bg-primary" :style="{ width: c.pct }" /></div>
        </div>
      </div>
      <span v-if="verdict" class="ml-10 mt-3 inline-flex items-center gap-1.5 rounded-full px-[9px] py-[3px] text-xs" :class="verdictClass[0]">
        <span class="size-1.5 rounded-full" :class="verdictClass[1]" />{{ verdict.label }}
      </span>
      <div v-if="open" class="ml-10 mt-3.5 flex flex-col gap-2 rounded-lg border border-default bg-default p-3 text-sm" @click.stop>
        <div v-for="p in r.scores" :key="p.user_id" class="lining grid grid-cols-[3rem_repeat(5,1fr)] gap-2 text-toned">
          <span class="truncate text-muted">{{ p.name }}</span>
          <span v-for="c in RATING_CRITERIA" :key="c.key">{{ formatScore(p[c.key]) }}</span>
        </div>
        <NuxtLink :to="`/r/${r.slug}`" class="pt-1 text-primary">View visits &amp; dishes →</NuxtLink>
      </div>
    </div>
  </div>
</template>
