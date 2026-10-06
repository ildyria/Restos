<script setup lang="ts">
import type { Photo, RestaurantDetail, Visit } from '~~/shared/types'

// Admin: a visit's photos. Upload first, then say which dish each one shows (its caption).
const props = defineProps<{ restaurant: RestaurantDetail, visit: Visit }>()
const emit = defineEmits<{ updated: [detail: RestaurantDetail] }>()

const toast = useToast()
const files = ref<File[]>([])
const uploading = ref(false)

const dishName = (p: Photo) => props.visit.dishes.find(d => d.id === p.dish_id)?.name ?? ''
// Uncaptioned first: they're the ones that still need a dish.
const photos = computed(() => [...props.visit.photos, ...props.visit.dishes.flatMap(d => d.photos)])
const listId = computed(() => `dishes-${props.visit.id}`)

const drafts = reactive<Record<number, string>>({})
watch(photos, (list) => {
  for (const p of list) drafts[p.id] = dishName(p)
}, { immediate: true })

watch(files, async (list) => {
  if (!list?.length) return
  uploading.value = true
  try {
    const fd = new FormData()
    for (const f of list) fd.append('files', f)
    emit('updated', await api<RestaurantDetail>(`/api/admin/visits/${props.visit.id}/photos`, { method: 'POST', body: fd }))
  } catch (e) {
    toast.add({ title: 'Upload failed', description: errorMessage(e), color: 'error' })
  } finally {
    files.value = []
    uploading.value = false
  }
})

const savingCaption = ref<number | null>(null)
async function saveCaption(p: Photo) {
  const next = (drafts[p.id] ?? '').trim()
  if (next === dishName(p)) return
  savingCaption.value = p.id
  try {
    emit('updated', await api<RestaurantDetail>(`/api/admin/photos/${p.id}`, { method: 'PUT', body: { dish: next } }))
  } catch (e) {
    drafts[p.id] = dishName(p)
    toast.add({ title: 'Could not save the dish', description: errorMessage(e), color: 'error' })
  } finally {
    savingCaption.value = null
  }
}

async function setCover(id: number) {
  try {
    emit('updated', await api<RestaurantDetail>(`/api/admin/restaurants/${props.restaurant.id}/cover`, { method: 'PUT', body: { photo_id: id } }))
  } catch (e) {
    toast.add({ title: 'Could not set cover', description: errorMessage(e), color: 'error' })
  }
}

const toDelete = ref<number | null>(null)
const confirmOpen = computed({ get: () => toDelete.value !== null, set: (v) => { if (!v) toDelete.value = null } })
async function deletePhoto() {
  if (toDelete.value === null) return
  try {
    emit('updated', await api<RestaurantDetail>(`/api/admin/photos/${toDelete.value}`, { method: 'DELETE' }))
    toDelete.value = null
  } catch (e) {
    toast.add({ title: 'Could not delete photo', description: errorMessage(e), color: 'error' })
  }
}
</script>

<template>
  <div class="space-y-3">
    <datalist :id="listId">
      <option v-for="d in visit.dishes" :key="d.id" :value="d.name" />
    </datalist>

    <ul v-if="photos.length" class="grid grid-cols-2 gap-3 sm:grid-cols-3">
      <li v-for="p in photos" :key="p.id" class="overflow-hidden rounded-md border border-default">
        <div class="group relative bg-elevated">
          <img :src="mediaUrl(p.key)" :alt="dishName(p) || 'Photo'" class="aspect-[4/3] w-full object-cover">
          <div class="absolute inset-x-0 top-0 flex justify-between gap-1 p-1.5">
            <UButton
              :icon="restaurant.cover_photo_id === p.id ? 'i-lucide-bookmark-check' : 'i-lucide-bookmark'"
              size="xs"
              :color="restaurant.cover_photo_id === p.id ? 'primary' : 'neutral'"
              :variant="restaurant.cover_photo_id === p.id ? 'solid' : 'subtle'"
              :aria-label="restaurant.cover_photo_id === p.id ? 'Current cover' : 'Use as cover'"
              :title="restaurant.cover_photo_id === p.id ? 'Current cover' : 'Use as cover'"
              @click="setCover(p.id)"
            />
            <UButton icon="i-lucide-trash-2" size="xs" color="error" variant="subtle" aria-label="Delete photo" @click="toDelete = p.id" />
          </div>
        </div>
        <UInput
          v-model="drafts[p.id]"
          :list="listId"
          placeholder="Which dish?"
          size="sm"
          variant="ghost"
          :loading="savingCaption === p.id"
          :aria-label="`Dish shown in photo ${p.id}`"
          class="w-full"
          @change="saveCaption(p)"
          @keydown.enter.prevent="($event.target as HTMLInputElement).blur()"
        />
      </li>
    </ul>

    <UFileUpload
      v-model="files"
      multiple
      accept="image/jpeg,image/png,image/webp"
      :label="photos.length ? 'Add more photos' : 'Drop photos here or click to browse'"
      description="JPEG, PNG or WebP, up to 20 MB each. Name the dish under each photo afterwards."
      :disabled="uploading"
      class="min-h-24"
    />
    <p v-if="uploading" class="text-sm text-muted">Uploading…</p>

    <ConfirmModal v-model:open="confirmOpen" title="Delete photo?" description="This can't be undone." @confirm="deletePhoto" />
  </div>
</template>
