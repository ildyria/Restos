<script setup lang="ts">
import Sortable from 'sortablejs'
import type { RestaurantSummary } from '~~/shared/types'

definePageMeta({ layout: 'admin' })
useSeoMeta({ title: `Ranking — ${SITE_NAME} admin`, robots: 'noindex' })

const toast = useToast()
const { data } = await useFetch<RestaurantSummary[]>('/api/restaurants', { query: { sort: 'rank' }, default: () => [] })

const ranked = ref<RestaurantSummary[]>([])
const unranked = ref<RestaurantSummary[]>([])
const rankedEl = useTemplateRef<HTMLElement>('rankedEl')
const unrankedEl = useTemplateRef<HTMLElement>('unrankedEl')
const dirty = ref(false)
const saving = ref(false)

function load() {
  ranked.value = (data.value ?? []).filter(r => r.rank_position !== null)
  unranked.value = (data.value ?? []).filter(r => r.rank_position === null)
  dirty.value = false
}
load()

// Sortable moves DOM nodes itself; on drop we undo that move and apply it to the arrays instead,
// so Vue stays the single source of truth (works within and between the two lists).
function onEnd(evt: Sortable.SortableEvent) {
  const { item, from, to, oldIndex, newIndex } = evt
  if (oldIndex === undefined || newIndex === undefined) return
  item.remove()
  from.insertBefore(item, from.children[oldIndex] ?? null)
  if (from === to && oldIndex === newIndex) return

  const lists = new Map<HTMLElement, Ref<RestaurantSummary[]>>([
    [rankedEl.value!, ranked],
    [unrankedEl.value!, unranked]
  ])
  const src = lists.get(from)!
  const dst = lists.get(to)!
  const srcItems = [...src.value]
  const [moved] = srcItems.splice(oldIndex, 1)
  if (from === to) {
    srcItems.splice(newIndex, 0, moved!)
    src.value = srcItems
  } else {
    const dstItems = [...dst.value]
    dstItems.splice(newIndex, 0, moved!)
    src.value = srcItems
    dst.value = dstItems
  }
  dirty.value = true
}

const sortables: Sortable[] = []
onMounted(() => {
  for (const el of [rankedEl.value, unrankedEl.value]) {
    if (el) sortables.push(Sortable.create(el, { animation: 150, group: 'ranking', handle: '.drag-handle', ghostClass: 'opacity-40', onEnd }))
  }
})
onBeforeUnmount(() => sortables.forEach(s => s.destroy()))

async function save() {
  saving.value = true
  try {
    await api('/api/admin/ranking', { method: 'PUT', body: { ids: ranked.value.map(r => r.id) } })
    dirty.value = false
    toast.add({ title: 'Ranking saved', color: 'success' })
  } catch (e) {
    toast.add({ title: 'Could not save ranking', description: errorMessage(e), color: 'error' })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UDashboardPanel>
    <template #header>
      <UDashboardNavbar title="Ranking">
        <template #right>
          <UButton v-if="dirty" color="neutral" variant="ghost" size="sm" @click="load">Discard</UButton>
          <UButton :disabled="!dirty" :loading="saving" size="sm" icon="i-lucide-check" @click="save">Save order</UButton>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <p class="mb-6 max-w-xl text-sm text-muted">
        Drag restaurants into the order you'd recommend them. Best at the top. Anything left under "Not ranked" appears below the list on the site.
      </p>
      <div class="grid gap-8 lg:grid-cols-2">
        <section>
          <h2 class="font-display mb-3 text-xl">Ranked</h2>
          <ol ref="rankedEl" class="min-h-16 space-y-2 rounded-lg border border-dashed border-default p-2">
            <li v-for="(r, i) in ranked" :key="r.id" class="flex items-center gap-3 rounded-md border border-default bg-default px-3 py-2">
              <UIcon name="i-lucide-grip-vertical" class="drag-handle size-5 shrink-0 cursor-grab text-dimmed" />
              <span class="rank-numeral w-7 text-xl text-primary">{{ i + 1 }}</span>
              <span class="flex-1 truncate font-medium">{{ r.name }}</span>
              <span class="text-sm tabular-nums text-muted">{{ formatScore(r.overall) }}</span>
            </li>
          </ol>
        </section>
        <section>
          <h2 class="font-display mb-3 text-xl">Not ranked</h2>
          <ul ref="unrankedEl" class="min-h-16 space-y-2 rounded-lg border border-dashed border-default p-2">
            <li v-for="r in unranked" :key="r.id" class="flex items-center gap-3 rounded-md border border-default bg-default px-3 py-2">
              <UIcon name="i-lucide-grip-vertical" class="drag-handle size-5 shrink-0 cursor-grab text-dimmed" />
              <span class="flex-1 truncate font-medium">{{ r.name }}</span>
              <span class="text-sm tabular-nums text-muted">{{ formatScore(r.overall) }}</span>
            </li>
          </ul>
        </section>
      </div>
    </template>
  </UDashboardPanel>
</template>
