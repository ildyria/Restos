<script setup lang="ts">
import type { RestaurantSummary } from '~~/shared/types'

defineProps<{ restaurant: RestaurantSummary, showRank?: boolean }>()
</script>

<template>
  <NuxtLink
    :to="`/r/${restaurant.slug}`"
    class="group grid grid-cols-[auto_1fr] items-center gap-x-5 gap-y-3 border-b border-default py-6 transition-colors hover:bg-elevated/50 sm:grid-cols-[3.5rem_9rem_1fr_auto] sm:px-2"
  >
    <span
      v-if="showRank"
      class="rank-numeral w-10 text-4xl text-primary sm:w-auto sm:text-5xl"
    >{{ restaurant.rank_position }}</span>
    <span v-else class="hidden sm:block" />

    <div class="aspect-[4/3] w-28 overflow-hidden rounded-md bg-elevated sm:w-36">
      <img
        v-if="restaurant.cover_key"
        :src="mediaUrl(restaurant.cover_key)"
        :alt="`${restaurant.name}`"
        class="size-full object-cover transition-transform duration-500 group-hover:scale-[1.03]"
        loading="lazy"
      >
      <div v-else class="grid size-full place-items-center text-dimmed">
        <UIcon name="i-lucide-utensils-crossed" class="size-6" />
      </div>
    </div>

    <div class="col-span-2 min-w-0 sm:col-span-1">
      <h3 class="font-display truncate text-xl font-medium">{{ restaurant.name }}</h3>
      <p class="mt-0.5 text-sm text-muted">
        <span v-if="restaurant.cuisine">{{ restaurant.cuisine }}</span>
        <span v-if="restaurant.cuisine && restaurant.city"> · </span>
        <span v-if="restaurant.city">{{ restaurant.city }}</span>
        <span v-if="restaurant.price_level"> · {{ priceLabel(restaurant.price_level) }}</span>
        <template v-if="restaurant.michelin_stars"> · <MichelinStars :count="restaurant.michelin_stars" /></template>
      </p>
      <p v-if="restaurant.last_visit" class="mt-1 text-xs text-dimmed">
        {{ restaurant.visit_count > 1 ? `${restaurant.visit_count} visits, last` : 'Visited' }}
        {{ formatDate(restaurant.last_visit, { month: 'short', year: 'numeric' }) }}
      </p>
    </div>

    <div class="col-span-2 flex items-center justify-between gap-4 sm:col-span-1 sm:flex-col sm:items-end sm:justify-center">
      <div class="flex items-baseline gap-2">
        <span class="font-display text-3xl tabular-nums">{{ formatScore(restaurant.overall) }}</span>
        <span class="text-xs text-dimmed">/ 5</span>
      </div>
      <RatingStars :value="restaurant.overall" :size="14" />
    </div>
  </NuxtLink>
</template>
