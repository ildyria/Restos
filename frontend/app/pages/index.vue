<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import type { RestaurantSummary } from '~~/shared/types'
import { RATING_CRITERIA } from '~~/shared/types'
import type { MapPin } from '~/components/RestaurantMap.vue'

useSeoMeta({ title: `${SITE_NAME} — where we ate`, description: 'Restaurants we have visited, ranked by preference.' })

const { data: list, status } = await useFetch<RestaurantSummary[]>('/api/restaurants', { query: { sort: 'rank' }, default: () => [] })

const ranked = computed(() => (list.value ?? []).filter(r => r.rank_position !== null))
const unranked = computed(() => (list.value ?? []).filter(r => r.rank_position === null))

const pins = computed<MapPin[]>(() => (list.value ?? [])
  .filter(r => r.latitude !== null && r.longitude !== null)
  .map(r => ({ id: r.id, label: r.rank_position ? String(r.rank_position) : '', name: r.name, lat: r.latitude!, lng: r.longitude! })))

// Side map from 1280px; below that the map is a tab next to the list.
const wide = useMediaQuery('(min-width: 1280px)')
const view = ref<'list' | 'map'>('list')
const showMap = computed(() => pins.value.length > 0)
const tabs = [{ value: 'list', label: 'List' }, { value: 'map', label: 'Map' }] as const

const hovered = ref<number | null>(null)
const opened = ref<number | null>(null)
const active = computed(() => hovered.value ?? opened.value)

const toggle = (id: number) => { opened.value = opened.value === id ? null : id }
const onHover = (id: number, on: boolean) => { hovered.value = on ? id : (hovered.value === id ? null : hovered.value) }

async function selectPin(id: number) {
  opened.value = id
  view.value = 'list'
  await nextTick()
  document.getElementById(`restaurant-${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}
</script>

<template>
  <div :class="showMap && wide ? 'grid grid-cols-[minmax(0,1fr)_400px] gap-10' : ''">
    <div class="min-w-0">
      <h1 class="sr-only">Where we ate</h1>
      <section v-if="showMap && !wide" class="mb-6 flex sm:justify-end">
        <div class="flex gap-0.5 rounded-lg border border-default bg-elevated p-[3px]" role="tablist">
          <button
            v-for="t in tabs"
            :key="t.value"
            type="button"
            role="tab"
            :aria-selected="view === t.value"
            class="rounded-md px-3 py-2 text-sm sm:py-1.5"
            :class="view === t.value ? 'bg-default text-highlighted shadow-xs' : 'text-muted'"
            @click="view = t.value"
          >
            {{ t.label }}
          </button>
        </div>
      </section>

      <div v-if="status === 'pending' && !list?.length" class="py-20 text-center text-dimmed">Loading…</div>

      <div v-else-if="!list?.length" class="py-20 text-center">
        <p class="font-display text-2xl">No restaurants yet.</p>
        <p class="mt-2 text-sm text-muted">The first entry will show up here once it has been added.</p>
      </div>

      <div v-else-if="showMap && !wide && view === 'map'" class="h-[560px] overflow-hidden rounded-lg border border-default">
        <RestaurantMap :pins="pins" :active="active" @hover="hovered = $event" @select="selectPin" />
      </div>

      <template v-else>
        <div class="hidden grid-cols-[2.75rem_6rem_minmax(0,1fr)_16.5rem_3.5rem] gap-5 border-b border-default px-3 pb-2 text-[11px] uppercase tracking-normal text-dimmed sm:grid">
          <span /><span /><span />
          <div class="grid grid-cols-4 gap-2"><span v-for="c in RATING_CRITERIA" :key="c.key">{{ c.label }}</span></div>
          <span class="text-right">Score</span>
        </div>
        <RestaurantRow
          v-for="r in ranked"
          :id="`restaurant-${r.id}`"
          :key="r.id"
          :restaurant="r"
          show-rank
          :open="opened === r.id"
          :highlighted="hovered === r.id"
          @toggle="toggle(r.id)"
          @hover="onHover(r.id, $event)"
        />
        <section v-if="unranked.length" class="mt-14">
          <h2 class="font-display text-xl text-muted">Not ranked yet</h2>
          <div class="mt-3 border-t border-default">
            <RestaurantRow
              v-for="r in unranked"
              :id="`restaurant-${r.id}`"
              :key="r.id"
              :restaurant="r"
              :open="opened === r.id"
              :highlighted="hovered === r.id"
              @toggle="toggle(r.id)"
              @hover="onHover(r.id, $event)"
            />
          </div>
        </section>
      </template>
    </div>

    <aside v-if="showMap && wide" class="-my-11 border-l border-default">
      <div class="sticky top-0 h-[min(100vh,760px)]">
        <RestaurantMap :pins="pins" :active="active" @hover="hovered = $event" @select="selectPin" />
      </div>
    </aside>
  </div>
</template>
