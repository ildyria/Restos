<script setup lang="ts">
import { z } from 'zod'
import { RATING_CRITERIA } from '~~/shared/types'
import type { Photo, RestaurantDetail, RestaurantInput, ScoreInput, SitePreview, User, Visit } from '~~/shared/types'

definePageMeta({ layout: 'admin' })

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
const users = await api<User[]>('/api/admin/users').catch(() => [] as User[])

useSeoMeta({ title: () => `${detail.value?.name ?? 'New restaurant'} — ${SITE_NAME} admin`, robots: 'noindex' })

// One score block per admin, prefilled with what they already entered.
function scoresFor(d: RestaurantDetail | null): ScoreInput[] {
  return users.map((u) => {
    const s = d?.scores.find(x => x.user_id === u.id)
    return {
      user_id: u.id,
      rating_food: s?.rating_food ?? null,
      rating_service: s?.rating_service ?? null,
      rating_ambiance: s?.rating_ambiance ?? null,
      rating_value: s?.rating_value ?? null,
      rating_quiet: s?.rating_quiet ?? null,
      would_return: s?.would_return ?? null
    }
  })
}
const blank = (): RestaurantInput => ({
  name: '', cuisine: '', address: '', city: '', website: '', price_level: null, michelin_stars: null,
  latitude: null, longitude: null, review: '', scores: scoresFor(null)
})
const state = reactive<RestaurantInput>(blank())
// New restaurants start with a visit (today, by default) so photos can go in right away, unless
// we don't remember when we went: then there's no visit until one is added.
const firstVisit = reactive({ visited_on: todayISO(), meal: mealNow(), notes: '' })
const dateForgotten = ref(false)
const meals = [{ label: 'Lunch', value: 'lunch' }, { label: 'Dinner', value: 'dinner' }]
function fillForm() {
  const d = detail.value
  Object.assign(state, d
    ? {
        name: d.name, cuisine: d.cuisine, address: d.address, city: d.city, website: d.website, price_level: d.price_level, michelin_stars: d.michelin_stars,
        latitude: d.latitude, longitude: d.longitude, review: d.review, scores: scoresFor(d)
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

const mean = (vals: (number | null)[]) => {
  const xs = vals.filter((v): v is number => v !== null)
  return xs.length ? xs.reduce((a, b) => a + b, 0) / xs.length : null
}
const personOverall = (s: ScoreInput) => mean(RATING_CRITERIA.map(c => s[c.key]))
// Same as the server: average each category over people, then average the categories.
const overall = computed(() => mean(RATING_CRITERIA.map(c => mean(state.scores.map(s => s[c.key])))))
const userName = (id: number) => users.find(u => u.id === id)?.name ?? `#${id}`
const returnItems = [
  { label: 'Not answered', value: 'unset' },
  { label: 'Would go back', value: 'yes' },
  { label: 'Wouldn’t go back', value: 'no' }
]
const getReturn = (s: ScoreInput) => s.would_return === null ? 'unset' : s.would_return ? 'yes' : 'no'
const setReturn = (s: ScoreInput, v: string) => { s.would_return = v === 'unset' ? null : v === 'yes' }

// Filling in the address places the map pin.
const locationPicker = useTemplateRef<{ autoLookup: () => void }>('locationPicker')

// Website picture: read as soon as the address is entered (no need to save first). The
// fetched picture is kept when the restaurant is saved. undefined = unchanged, '' = removed.
const siteKey = ref<string | undefined>()
const fetchingSite = ref(false)
const siteNote = ref('')
let lastFetched = detail.value?.website ?? ''
const sitePictureKey = computed(() => siteKey.value === undefined ? (detail.value?.site_photo?.key ?? null) : (siteKey.value || null))

async function readWebsite(force = false) {
  const url = state.website.trim()
  if (!/^https?:\/\/[^\s/]+\.\S+/.test(url) || (url === lastFetched && !force)) return
  lastFetched = url
  fetchingSite.value = true
  siteNote.value = ''
  try {
    const p = await api<SitePreview>('/api/admin/site-preview', { method: 'POST', body: { url } })
    if (p.title && !state.name.trim()) state.name = p.title
    if (p.image_key) siteKey.value = p.image_key
    siteNote.value = p.note ?? ''
  } catch (e) {
    siteNote.value = errorMessage(e)
  } finally {
    fetchingSite.value = false
  }
}

// Cover: the restaurant photo, the website picture or any visit photo; none chosen = automatic
// (restaurant photo first, then the website picture, then the latest visit's first photo).
const coverCandidates = computed<Photo[]>(() => {
  const d = detail.value
  if (!d) return []
  const visitPhotos = d.visits.flatMap(v => [...v.photos, ...v.dishes.flatMap(x => x.photos)])
  return [d.restaurant_photo, d.site_photo, ...visitPhotos].filter((p): p is Photo => !!p)
})
const coverBadge = { restaurant: 'Restaurant', website: 'Website', upload: '' } as const
const coverLabel = {
  restaurant: 'Use the restaurant photo as cover',
  website: 'Use the website picture as cover',
  upload: 'Use this photo as cover'
} as const

// The restaurant's own photo (the place itself, not a dish): uploading one replaces the previous
// one and makes it the cover.
const photoInput = ref<HTMLInputElement | null>(null)
const uploadingPhoto = ref(false)
async function uploadRestaurantPhoto(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  uploadingPhoto.value = true
  try {
    const fd = new FormData()
    fd.append('file', file)
    detail.value = await api<RestaurantDetail>(`/api/admin/restaurants/${detail.value!.id}/photo`, { method: 'POST', body: fd })
  } catch (err) {
    toast.add({ title: 'Upload failed', description: errorMessage(err), color: 'error' })
  } finally {
    input.value = ''
    uploadingPhoto.value = false
  }
}
async function chooseCover(photoId: number | null) {
  try {
    detail.value = await api<RestaurantDetail>(`/api/admin/restaurants/${detail.value!.id}/cover`, { method: 'PUT', body: { photo_id: photoId } })
  } catch (e) {
    toast.add({ title: 'Could not set cover', description: errorMessage(e), color: 'error' })
  }
}

const saving = ref(false)
async function save() {
  saving.value = true
  try {
    if (isNew.value) {
      const visit = dateForgotten.value ? undefined : { ...firstVisit }
      const created = await api<RestaurantDetail>('/api/admin/restaurants', { method: 'POST', body: { ...state, first_visit: visit, site_image_key: siteKey.value } })
      toast.add({ title: 'Restaurant created', description: visit && 'Now add the photos from your visit.', color: 'success' })
      await navigateTo(`/admin/restaurants/${created.id}`)
    } else {
      detail.value = await api<RestaurantDetail>(`/api/admin/restaurants/${detail.value!.id}`, { method: 'PUT', body: { ...state, site_image_key: siteKey.value } })
      siteKey.value = undefined
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
const deleteRestaurantPhoto = (p: Photo) => ask('Remove the restaurant photo?', 'The cover goes back to the next available picture.',
  () => api<RestaurantDetail>(`/api/admin/photos/${p.id}`, { method: 'DELETE' }))
const deleteDish = (id: number, name: string) => ask(`Delete "${name}"?`, 'Its photos stay on the visit, without a dish.',
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
            <UFormField label="Address" name="address"><UInput v-model="state.address" class="w-full" @change="locationPicker?.autoLookup()" /></UFormField>
            <UFormField label="City" name="city"><UInput v-model="state.city" class="w-full" @change="locationPicker?.autoLookup()" /></UFormField>
            <UFormField label="Website" name="website" class="sm:col-span-2" help="Its preview picture and name are read as soon as you leave this field.">
              <UInput v-model="state.website" type="url" placeholder="https://" :loading="fetchingSite" class="w-full" @change="readWebsite()" />
            </UFormField>
            <div v-if="sitePictureKey || fetchingSite || siteNote" class="flex items-start gap-4 sm:col-span-2">
              <div class="aspect-[1.91/1] w-48 shrink-0 overflow-hidden rounded-md border border-default bg-elevated">
                <img v-if="sitePictureKey" :src="mediaUrl(sitePictureKey)" alt="Picture from the website" class="size-full object-cover">
                <div v-else class="grid size-full place-items-center text-dimmed">
                  <UIcon :name="fetchingSite ? 'i-lucide-loader-circle' : 'i-lucide-image-off'" class="size-5" :class="fetchingSite && 'animate-spin'" />
                </div>
              </div>
              <div class="space-y-2 text-sm">
                <p class="font-medium">Picture from the website</p>
                <p v-if="fetchingSite" class="text-muted">Reading the website…</p>
                <p v-else-if="siteNote" class="text-muted">{{ siteNote }}</p>
                <p v-else-if="siteKey" class="text-muted">Kept when you save.</p>
                <div class="flex gap-1">
                  <UButton size="xs" color="neutral" variant="subtle" icon="i-lucide-refresh-cw" :disabled="fetchingSite || !state.website" @click="readWebsite(true)">Fetch again</UButton>
                  <UButton v-if="sitePictureKey" size="xs" color="neutral" variant="ghost" icon="i-lucide-x" @click="siteKey = ''">Remove</UButton>
                </div>
              </div>
            </div>
            <template v-if="isNew">
              <template v-if="!dateForgotten">
                <UFormField label="First visit" name="first_visit">
                  <DateInput v-model="firstVisit.visited_on" />
                </UFormField>
                <UFormField label="Meal" name="first_visit_meal">
                  <USelect v-model="firstVisit.meal" :items="meals" class="w-full" />
                </UFormField>
              </template>
              <UCheckbox v-model="dateForgotten" label="We don't remember when we went" class="sm:col-span-2" />
            </template>
            <UFormField label="On the map" name="latitude" class="sm:col-span-2">
              <LocationPicker ref="locationPicker" v-model:lat="state.latitude" v-model:lng="state.longitude" :address="state.address" :city="state.city" />
            </UFormField>
          </section>

          <section>
            <div class="mb-3 flex items-baseline justify-between">
              <h2 class="font-display text-xl">Ratings</h2>
              <p class="text-sm text-muted">Overall <span class="font-display text-lg text-default tabular-nums">{{ formatScore(overall) }}</span></p>
            </div>
            <div class="space-y-4">
              <div v-for="s in state.scores" :key="s.user_id" class="rounded-lg border border-default p-4">
                <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
                  <p class="font-medium">{{ userName(s.user_id) }} <span class="ml-1 text-sm font-normal tabular-nums text-muted">{{ formatScore(personOverall(s)) }}</span></p>
                  <USelect :model-value="getReturn(s)" @update:model-value="setReturn(s, $event)" :items="returnItems" size="sm" class="w-44" :aria-label="`Would ${userName(s.user_id)} go back?`" />
                </div>
                <div class="grid gap-3 sm:grid-cols-2">
                  <div v-for="c in RATING_CRITERIA" :key="c.key">
                    <p class="mb-1 text-sm text-muted">{{ c.label }}</p>
                    <RatingInput v-model="s[c.key]" :label="`${userName(s.user_id)}: ${c.label}`" />
                  </div>
                </div>
              </div>
            </div>
            <UFormField label="Review" name="review" class="mt-4" help="Shown on the restaurant page.">
              <UTextarea v-model="state.review" :rows="5" autoresize class="w-full" />
            </UFormField>
          </section>
        </UForm>

        <section v-if="detail">
          <div class="mb-3 flex items-baseline justify-between">
            <h2 class="font-display text-xl">Cover</h2>
            <p class="text-sm text-muted">Shown in the list and at the top of the restaurant page.</p>
          </div>
          <div class="mb-4 flex items-center gap-4">
            <div class="grid aspect-[4/3] w-28 shrink-0 place-items-center overflow-hidden rounded-md bg-elevated text-dimmed">
              <img v-if="detail.restaurant_photo" :src="mediaUrl(detail.restaurant_photo.key)" alt="Photo of the restaurant" class="size-full object-cover">
              <UIcon v-else name="i-lucide-store" class="size-5" />
            </div>
            <div class="space-y-2 text-sm">
              <p class="font-medium">Photo of the restaurant</p>
              <p class="text-muted">The place itself rather than a dish. Uploading one makes it the cover.</p>
              <div class="flex gap-2">
                <input ref="photoInput" type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="uploadRestaurantPhoto">
                <UButton size="xs" color="neutral" variant="subtle" icon="i-lucide-upload" :loading="uploadingPhoto" @click="photoInput?.click()">
                  {{ detail.restaurant_photo ? 'Replace' : 'Upload photo' }}
                </UButton>
                <UButton v-if="detail.restaurant_photo" size="xs" color="neutral" variant="ghost" icon="i-lucide-x" @click="deleteRestaurantPhoto(detail.restaurant_photo)">Remove</UButton>
              </div>
            </div>
          </div>
          <ul v-if="coverCandidates.length" class="grid grid-cols-3 gap-3 sm:grid-cols-5">
            <li>
              <button
                type="button"
                class="grid aspect-[4/3] w-full place-items-center rounded-md border p-2 text-center text-xs"
                :class="detail.cover_photo_id === null ? 'border-primary ring-2 ring-primary' : 'border-default text-muted hover:border-accented'"
                :aria-pressed="detail.cover_photo_id === null"
                @click="chooseCover(null)"
              >
                <span><UIcon name="i-lucide-wand-sparkles" class="mb-1 size-4" /><br>Automatic</span>
              </button>
            </li>
            <li v-for="p in coverCandidates" :key="p.id" class="relative">
              <button
                type="button"
                class="block aspect-[4/3] w-full overflow-hidden rounded-md border"
                :class="detail.cover_photo_id === p.id ? 'border-primary ring-2 ring-primary' : 'border-default hover:border-accented'"
                :aria-pressed="detail.cover_photo_id === p.id"
                :aria-label="coverLabel[p.source]"
                @click="chooseCover(p.id)"
              >
                <img :src="mediaUrl(p.key)" alt="" class="size-full object-cover">
              </button>
              <span v-if="coverBadge[p.source]" class="pointer-events-none absolute bottom-1 left-1 rounded bg-black/60 px-1.5 py-0.5 text-[10px] text-white">{{ coverBadge[p.source] }}</span>
            </li>
          </ul>
          <p v-if="siteKey" class="mt-2 text-xs text-muted">Save to use the newly fetched website picture.</p>
        </section>

        <section v-if="detail">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="font-display text-xl">Visits</h2>
            <UButton icon="i-lucide-plus" size="sm" color="neutral" variant="subtle" @click="openVisit(null)">Add visit</UButton>
          </div>
          <p v-if="!detail.visits.length" class="rounded-lg border border-dashed border-default p-8 text-center text-sm text-muted">
            No visits yet. Add the day you went, then its photos.
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
              <div class="p-4">
                <VisitPhotos :restaurant="detail" :visit="v" @updated="onDetail" />
              </div>
              <p v-if="v.dishes.length" class="border-t border-default px-4 pb-1 pt-3 text-xs uppercase tracking-wide text-dimmed">Dishes</p>
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
        <p v-else-if="dateForgotten" class="text-sm text-muted">The restaurant starts without a visit; you can add one later.</p>
        <p v-else class="text-sm text-muted">Saving creates the first visit; you can add its photos right after.</p>
      </div>

      <VisitModal v-if="detail" v-model:open="visitOpen" :restaurant-id="detail.id" :visit="editingVisit" @saved="onDetail" />
      <DishEditor v-if="detail" v-model:open="dishOpen" :restaurant="detail" :visit-id="dishVisitId" :dish-id="dishId" @updated="onDetail" />
      <ConfirmModal v-model:open="confirm.open" :title="confirm.title" :description="confirm.description" :loading="confirm.loading" @confirm="confirm.action" />
    </template>
  </UDashboardPanel>
</template>
