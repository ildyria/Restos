<script setup lang="ts">
import { refDebounced } from '@vueuse/core'
import type { RestaurantSummary, SortMode } from '~~/shared/types'

useSeoMeta({ title: 'Restos — our list', description: 'Restaurants we have visited, ranked by preference.' })

const sort = ref<SortMode>('rank')
const search = ref('')
const cuisine = ref('all')
const q = refDebounced(search, 250)

const sorts: { label: string, value: SortMode }[] = [
  { label: 'Our ranking', value: 'rank' },
  { label: 'Top rated', value: 'rating' },
  { label: 'Recent', value: 'recent' },
  { label: 'A–Z', value: 'name' }
]

const query = computed(() => ({
  sort: sort.value,
  q: q.value || undefined,
  cuisine: cuisine.value === 'all' ? undefined : cuisine.value
}))

const { data: list, status } = await useFetch<RestaurantSummary[]>('/api/restaurants', { query, default: () => [] })
const { data: cuisines } = await useFetch<string[]>('/api/cuisines', { default: () => [] })

const cuisineItems = computed(() => [
  { label: 'All cuisines', value: 'all' },
  ...(cuisines.value ?? []).map(c => ({ label: c, value: c }))
])

const ranked = computed(() => (list.value ?? []).filter(r => r.rank_position !== null))
const unranked = computed(() => (list.value ?? []).filter(r => r.rank_position === null))
const byRank = computed(() => sort.value === 'rank')
const filtering = computed(() => !!q.value || cuisine.value !== 'all')
</script>

<template>
  <div>
    <section class="mb-10 max-w-2xl">
      <h1 class="font-display text-4xl font-medium leading-tight sm:text-5xl">
        Where we've eaten,<br><em class="text-primary">and how it went.</em>
      </h1>
      <p class="mt-4 text-muted">
        Every lunch and dinner worth remembering, ranked by how much we'd go back.
      </p>
    </section>

    <div class="mb-2 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex flex-wrap gap-1" role="group" aria-label="Sort">
        <UButton
          v-for="s in sorts"
          :key="s.value"
          size="sm"
          :color="sort === s.value ? 'primary' : 'neutral'"
          :variant="sort === s.value ? 'soft' : 'ghost'"
          @click="sort = s.value"
        >
          {{ s.label }}
        </UButton>
      </div>
      <div class="flex gap-2">
        <UInput v-model="search" icon="i-lucide-search" placeholder="Search" size="sm" class="w-40 sm:w-48" />
        <USelect v-if="(cuisines?.length ?? 0) > 0" v-model="cuisine" :items="cuisineItems" size="sm" class="w-40" />
      </div>
    </div>

    <div v-if="status === 'pending' && !list?.length" class="py-20 text-center text-dimmed">Loading…</div>

    <div v-else-if="!list?.length" class="py-20 text-center">
      <p class="font-display text-2xl">{{ filtering ? 'Nothing matches.' : 'No restaurants yet.' }}</p>
      <p class="mt-2 text-sm text-muted">
        {{ filtering ? 'Try a different search or cuisine.' : 'The first entry will show up here once it has been added.' }}
      </p>
    </div>

    <template v-else-if="byRank">
      <div class="border-t border-default">
        <RestaurantRow v-for="r in ranked" :key="r.id" :restaurant="r" show-rank />
      </div>
      <section v-if="unranked.length" class="mt-14">
        <h2 class="font-display text-xl text-muted">Not ranked yet</h2>
        <div class="mt-3 border-t border-default">
          <RestaurantRow v-for="r in unranked" :key="r.id" :restaurant="r" />
        </div>
      </section>
    </template>

    <div v-else class="border-t border-default">
      <RestaurantRow v-for="r in list" :key="r.id" :restaurant="r" />
    </div>
  </div>
</template>
