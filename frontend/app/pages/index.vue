<script setup lang="ts">
import { onKeyStroke, useMediaQuery } from '@vueuse/core'
import type { RestaurantSummary } from '~~/shared/types'
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
// The map zooms to a restaurant hovered in the list; hovering a pin on the map doesn't move it.
const focused = ref<number | null>(null)

const toggle = (id: number) => { opened.value = opened.value === id ? null : id }
const onHover = (id: number, on: boolean) => {
  hovered.value = on ? id : (hovered.value === id ? null : hovered.value)
  focused.value = on ? id : (focused.value === id ? null : focused.value)
}

// Hovering a pin scrolls its row into view (when it's off-screen), after a short pause so that
// sweeping across the map doesn't make the page jump.
let pinTimer: ReturnType<typeof setTimeout> | undefined
function onPinHover(id: number | null) {
  hovered.value = id
  clearTimeout(pinTimer)
  if (id !== null && !expanded.value) pinTimer = setTimeout(() => revealRow(id), 250)
}
function revealRow(id: number) {
  const row = document.getElementById(`restaurant-${id}`)
  if (!row) return // the list isn't shown (map tab)
  const box = row.getBoundingClientRect()
  if (box.top >= 0 && box.bottom <= window.innerHeight) return
  const smooth = !window.matchMedia('(prefers-reduced-motion: reduce)').matches
  row.scrollIntoView({ behavior: smooth ? 'smooth' : 'auto', block: 'center' })
}
onBeforeUnmount(() => clearTimeout(pinTimer))

// Clicking the side map widens it over the list; under it, only the restaurant whose pin is hovered.
const expanded = ref(false)
const preview = computed(() => expanded.value ? (list.value ?? []).find(r => r.id === hovered.value) : undefined)
let listScroll = 0
async function expandMap() {
  if (!wide.value || expanded.value) return
  listScroll = window.scrollY
  expanded.value = true
  hovered.value = focused.value = null
  await nextTick()
  window.scrollTo({ top: 0 })
}
async function collapseMap() {
  if (!expanded.value) return
  expanded.value = false
  await nextTick()
  window.scrollTo({ top: listScroll })
}
onKeyStroke('Escape', collapseMap)
watch(wide, (w) => { if (!w) expanded.value = false })

async function selectPin(id: number) {
  expanded.value = false
  opened.value = id
  view.value = 'list'
  await nextTick()
  document.getElementById(`restaurant-${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}
</script>

<template>
  <div :class="showMap && wide && !expanded ? 'grid grid-cols-[minmax(0,1fr)_400px] gap-10' : ''">
    <div v-show="!expanded" class="min-w-0">
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
        <RestaurantMap :pins="pins" :active="active" @hover="onPinHover" @select="selectPin" />
      </div>

      <template v-else>
        <RestaurantListHeader />
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

    <!-- The same map element in both layouts, so widening it keeps the visitor's zoom. -->
    <aside v-if="showMap && wide" :class="!expanded && '-my-11 border-l border-default'">
      <div
        class="relative"
        :class="expanded ? 'h-[min(75vh,680px)] overflow-hidden rounded-lg border border-default' : 'sticky top-0 h-[min(100vh,760px)]'"
      >
        <RestaurantMap :pins="pins" :active="active" :focus="focused" @hover="onPinHover" @select="selectPin" @map-click="expandMap" />
        <UButton
          class="absolute right-3 top-3 z-[1000] shadow-sm"
          color="neutral"
          variant="outline"
          size="sm"
          :icon="expanded ? 'i-lucide-minimize-2' : 'i-lucide-maximize-2'"
          :aria-label="expanded ? 'Back to the list' : 'Widen the map'"
          @click="expanded ? collapseMap() : expandMap()"
        >
          <template v-if="expanded">Back to the list</template>
        </UButton>
      </div>
      <div v-if="preview" class="mt-6">
        <RestaurantListHeader />
        <RestaurantRow :restaurant="preview" :show-rank="preview.rank_position !== null" />
      </div>
    </aside>
  </div>
</template>
