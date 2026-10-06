<script setup lang="ts">
import { z } from 'zod'
import { RATING_CRITERIA } from '~~/shared/types'
import type { RestaurantDetail, RestaurantInput, Visit } from '~~/shared/types'

definePageMeta({ layout: 'admin', middleware: 'admin' })

const route = useRoute()
const toast = useToast()
const isNew = computed(() => route.params.id === 'new')

const detail = ref<RestaurantDetail | null>(null)
if (!isNew.value) {
  try {
    detail.value = await api<RestaurantDetail>(`/api/admin/restaurants/${route.params.id}`)
  } catch {
    throw createError({ statusCode: 404, statusMessage: 'Restaurant not found', fatal: true })
  }
}
useSeoMeta({ title: () => `${detail.value?.name ?? 'New restaurant'} — Restos admin`, robots: 'noindex' })

const blank = (): RestaurantInput => ({
  name: '', cuisine: '', address: '', city: '', website: '', price_level: null, michelin_stars: null,
  rating_food: null, rating_service: null, rating_ambiance: null, rating_value: null, review: ''
})
const state = reactive<RestaurantInput>(blank())
function fillForm() {
  const d = detail.value
  Object.assign(state, d
    ? {
        name: d.name, cuisine: d.cuisine, address: d.address, city: d.city, website: d.website, price_level: d.price_level, michelin_stars: d.michelin_stars,
        rating_food: d.rating_food, rating_service: d.rating_service, rating_ambiance: d.rating_ambiance,
        rating_value: d.rating_value, review: d.review
      }
    : blank())
}
fillForm()

const schema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(200),
  website: z.string().trim().refine(v => v === '' || /^https?:\/\//.test(v), 'Must start with http:// or https://')
})

const priceItems = [
  { label: 'Not set', value: 0 },
  { label: '€  — inexpensive', value: 1 },
  { label: '€€  — moderate', value: 2 },
  { label: '€€€  — pricey', value: 3 },
  { label: '€€€€  — splurge', value: 4 }
]
const price = computed({
  get: () => state.price_level ?? 0,
  set: (v: number) => { state.price_level = v === 0 ? null : v }
})

const starItems = [
  { label: 'None', value: 0 },
  { label: '1 star', value: 1 },
  { label: '2 stars', value: 2 },
  { label: '3 stars', value: 3 }
]
const stars = computed({
  get: () => state.michelin_stars ?? 0,
  set: (v: number) => { state.michelin_stars = v === 0 ? null : v }
})

const overall = computed(() => {
  const vals = RATING_CRITERIA.map(c => state[c.key]).filter((v): v is number => v !== null)
  return vals.length ? vals.reduce((a, b) => a + b, 0) / vals.length : null
})

const saving = ref(false)
async function save() {
  saving.value = true
  try {
    if (isNew.value) {
      const created = await api<RestaurantDetail>('/api/admin/restaurants', { method: 'POST', body: { ...state } })
      toast.add({ title: 'Restaurant created', description: 'Now add a visit and some dishes.', color: 'success' })
      await navigateTo(`/admin/restaurants/${created.id}`)
    } else {
      detail.value = await api<RestaurantDetail>(`/api/admin/restaurants/${detail.value!.id}`, { method: 'PUT', body: { ...state } })
      toast.add({ title: 'Saved', color: 'success' })
    }
  } catch (e) {
    toast.add({ title: 'Could not save', description: errorMessage(e), color: 'error' })
  } finally {
    saving.value = false
  }
}

// Visits & dishes. Every mutation returns the fresh restaurant, which replaces `detail`.
const onDetail = (d: RestaurantDetail) => { detail.value = d }

const visitOpen = ref(false)
const editingVisit = ref<Visit | null>(null)
function openVisit(v: Visit | null) { editingVisit.value = v; visitOpen.value = true }

const dishOpen = ref(false)
const dishVisitId = ref<number | null>(null)
const dishId = ref<number | null>(null)
function openDish(visitId: number, id: number | null) { dishVisitId.value = visitId; dishId.value = id; dishOpen.value = true }

const confirm = reactive({ open: false, title: '', description: '', loading: false, action: async () => {} })
function ask(title: string, description: string, action: () => Promise<RestaurantDetail>) {
  Object.assign(confirm, {
    open: true, title, description,
    action: async () => {
      confirm.loading = true
      try {
        detail.value = await action()
        confirm.open = false
      } catch (e) {
        toast.add({ title: 'Could not delete', description: errorMessage(e), color: 'error' })
      } finally {
        confirm.loading = false
      }
    }
  })
}
const deleteVisit = (v: Visit) => ask('Delete this visit?', `${formatDate(v.visited_on)} and all its dishes and photos will be removed.`,
  () => api<RestaurantDetail>(`/api/admin/visits/${v.id}`, { method: 'DELETE' }))
const deleteDish = (id: number, name: string) => ask(`Delete "${name}"?`, 'The dish and its photos will be removed.',
  () => api<RestaurantDetail>(`/api/admin/dishes/${id}`, { method: 'DELETE' }))
</script>

<template>
  <UDashboardPanel>
    <template #header>
      <UDashboardNavbar :title="isNew ? 'New restaurant' : (detail?.name ?? '')">
        <template #right>
          <UButton v-if="detail" :to="`/r/${detail.slug}`" color="neutral" variant="ghost" size="sm" icon="i-lucide-eye">View</UButton>
          <UButton size="sm" icon="i-lucide-check" :loading="saving" @click="($refs.form as any)?.submit()">Save</UButton>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="max-w-3xl space-y-12">
        <UForm ref="form" :state="state" :schema="schema" class="space-y-8" @submit="save">
          <section class="grid gap-4 sm:grid-cols-2">
            <UFormField label="Name" name="name" required class="sm:col-span-2">
              <UInput v-model="state.name" size="lg" class="w-full" />
            </UFormField>
            <UFormField label="Cuisine" name="cuisine"><UInput v-model="state.cuisine" placeholder="e.g. Neapolitan pizza" class="w-full" /></UFormField>
            <UFormField label="Price" name="price_level"><USelect v-model="price" :items="priceItems" class="w-full" /></UFormField>
            <UFormField label="Michelin stars" name="michelin_stars"><USelect v-model="stars" :items="starItems" class="w-full" /></UFormField>
            <UFormField label="Address" name="address"><UInput v-model="state.address" class="w-full" /></UFormField>
            <UFormField label="City" name="city"><UInput v-model="state.city" class="w-full" /></UFormField>
            <UFormField label="Website" name="website" class="sm:col-span-2"><UInput v-model="state.website" type="url" placeholder="https://" class="w-full" /></UFormField>
          </section>

          <section>
            <div class="mb-3 flex items-baseline justify-between">
              <h2 class="font-display text-xl">Rating</h2>
              <p class="text-sm text-muted">Overall <span class="font-display text-lg text-default tabular-nums">{{ formatScore(overall) }}</span></p>
            </div>
            <div class="grid gap-3 rounded-lg border border-default p-4 sm:grid-cols-2">
              <div v-for="c in RATING_CRITERIA" :key="c.key">
                <p class="mb-1 text-sm text-muted">{{ c.label }}</p>
                <RatingInput v-model="state[c.key]" :label="c.label" />
              </div>
            </div>
            <UFormField label="Review" name="review" class="mt-4" help="Shown on the restaurant page.">
              <UTextarea v-model="state.review" :rows="5" autoresize class="w-full" />
            </UFormField>
          </section>
        </UForm>

        <section v-if="detail">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="font-display text-xl">Visits</h2>
            <UButton icon="i-lucide-plus" size="sm" color="neutral" variant="subtle" @click="openVisit(null)">Add visit</UButton>
          </div>
          <p v-if="!detail.visits.length" class="rounded-lg border border-dashed border-default p-8 text-center text-sm text-muted">
            No visits yet. Add the day you went, then the dishes you ordered.
          </p>
          <ul class="space-y-4">
            <li v-for="v in detail.visits" :key="v.id" class="rounded-lg border border-default">
              <div class="flex items-center justify-between gap-2 border-b border-default px-4 py-3">
                <div class="flex items-center gap-2">
                  <span class="font-medium">{{ formatDate(v.visited_on) }}</span>
                  <UBadge :label="v.meal === 'lunch' ? 'Lunch' : 'Dinner'" color="neutral" variant="subtle" size="sm" />
                </div>
                <div class="flex">
                  <UButton icon="i-lucide-pencil" color="neutral" variant="ghost" size="xs" aria-label="Edit visit" @click="openVisit(v)" />
                  <UButton icon="i-lucide-trash-2" color="error" variant="ghost" size="xs" aria-label="Delete visit" @click="deleteVisit(v)" />
                </div>
              </div>
              <p v-if="v.notes" class="px-4 pt-3 text-sm text-muted">{{ v.notes }}</p>
              <ul class="divide-y divide-default">
                <li v-for="d in v.dishes" :key="d.id" class="flex items-center gap-3 px-4 py-3">
                  <div class="flex -space-x-2">
                    <img
                      v-for="p in d.photos.slice(0, 3)"
                      :key="p.id"
                      :src="mediaUrl(p.key)"
                      :alt="d.name"
                      class="size-10 rounded-md object-cover ring-2 ring-default"
                    >
                    <span v-if="!d.photos.length" class="grid size-10 place-items-center rounded-md bg-elevated text-dimmed">
                      <UIcon name="i-lucide-image-off" class="size-4" />
                    </span>
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm font-medium">{{ d.name }}</p>
                    <p class="text-xs text-dimmed">{{ [COURSES.find(c => c.value === d.course)?.label, `${d.photos.length} photo${d.photos.length === 1 ? '' : 's'}`].filter(Boolean).join(' · ') }}</p>
                  </div>
                  <UButton icon="i-lucide-pencil" color="neutral" variant="ghost" size="xs" aria-label="Edit dish" @click="openDish(v.id, d.id)" />
                  <UButton icon="i-lucide-trash-2" color="error" variant="ghost" size="xs" aria-label="Delete dish" @click="deleteDish(d.id, d.name)" />
                </li>
              </ul>
              <div class="px-4 py-2">
                <UButton icon="i-lucide-plus" size="xs" color="neutral" variant="link" @click="openDish(v.id, null)">Add dish</UButton>
              </div>
            </li>
          </ul>
        </section>
        <p v-else class="text-sm text-muted">Save the restaurant to start adding visits, dishes and photos.</p>
      </div>

      <VisitModal v-if="detail" v-model:open="visitOpen" :restaurant-id="detail.id" :visit="editingVisit" @saved="onDetail" />
      <DishEditor v-if="detail" v-model:open="dishOpen" :restaurant="detail" :visit-id="dishVisitId" :dish-id="dishId" @updated="onDetail" />
      <ConfirmModal v-model:open="confirm.open" :title="confirm.title" :description="confirm.description" :loading="confirm.loading" @confirm="confirm.action" />
    </template>
  </UDashboardPanel>
</template>
