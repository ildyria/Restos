<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { RestaurantSummary } from '~~/shared/types'

definePageMeta({ layout: 'admin', middleware: 'admin' })
useSeoMeta({ title: `Restaurants — ${SITE_NAME} admin`, robots: 'noindex' })

const toast = useToast()
const search = ref('')
const { data, refresh } = await useFetch<RestaurantSummary[]>('/api/restaurants', { query: { sort: 'name' }, default: () => [] })

const rows = computed(() => {
  const q = search.value.trim().toLowerCase()
  return (data.value ?? []).filter(r => !q || `${r.name} ${r.cuisine} ${r.city}`.toLowerCase().includes(q))
})

const columns: TableColumn<RestaurantSummary>[] = [
  { accessorKey: 'name', header: 'Name' },
  { accessorKey: 'cuisine', header: 'Cuisine' },
  { accessorKey: 'city', header: 'City' },
  { accessorKey: 'overall', header: 'Score' },
  { accessorKey: 'rank_position', header: 'Rank' },
  { accessorKey: 'visit_count', header: 'Visits' },
  { id: 'actions', header: '' }
]

const toDelete = ref<RestaurantSummary | null>(null)
const confirmOpen = computed({ get: () => !!toDelete.value, set: (v) => { if (!v) toDelete.value = null } })
const deleting = ref(false)

async function remove() {
  if (!toDelete.value) return
  deleting.value = true
  try {
    await api(`/api/admin/restaurants/${toDelete.value.id}`, { method: 'DELETE' })
    toast.add({ title: `Deleted ${toDelete.value.name}`, color: 'success' })
    toDelete.value = null
    await refresh()
  } catch (e) {
    toast.add({ title: 'Could not delete', description: errorMessage(e), color: 'error' })
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <UDashboardPanel>
    <template #header>
      <UDashboardNavbar title="Restaurants">
        <template #right>
          <UInput v-model="search" icon="i-lucide-search" placeholder="Filter" size="sm" class="w-40" />
          <UButton to="/admin/restaurants/new" icon="i-lucide-plus" size="sm">New restaurant</UButton>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <UTable :data="rows" :columns="columns" class="w-full">
        <template #name-cell="{ row }">
          <NuxtLink :to="`/admin/restaurants/${row.original.id}`" class="font-medium hover:underline">{{ row.original.name }}</NuxtLink>
        </template>
        <template #overall-cell="{ row }">{{ formatScore(row.original.overall) }}</template>
        <template #rank_position-cell="{ row }">{{ row.original.rank_position ?? '—' }}</template>
        <template #actions-cell="{ row }">
          <div class="flex justify-end gap-1">
            <UButton :to="`/r/${row.original.slug}`" icon="i-lucide-eye" color="neutral" variant="ghost" size="xs" aria-label="View on site" />
            <UButton :to="`/admin/restaurants/${row.original.id}`" icon="i-lucide-pencil" color="neutral" variant="ghost" size="xs" aria-label="Edit" />
            <UButton icon="i-lucide-trash-2" color="error" variant="ghost" size="xs" aria-label="Delete" @click="toDelete = row.original" />
          </div>
        </template>
      </UTable>
      <p v-if="!rows.length" class="py-12 text-center text-sm text-muted">
        {{ data?.length ? 'No match.' : 'No restaurants yet — add the first one.' }}
      </p>

      <UModal v-model:open="confirmOpen" title="Delete restaurant?" :description="`This permanently removes ${toDelete?.name} with all its visits, dishes and photos.`">
        <template #footer>
          <div class="flex w-full justify-end gap-2">
            <UButton color="neutral" variant="ghost" @click="toDelete = null">Cancel</UButton>
            <UButton color="error" :loading="deleting" @click="remove">Delete</UButton>
          </div>
        </template>
      </UModal>
    </template>
  </UDashboardPanel>
</template>
