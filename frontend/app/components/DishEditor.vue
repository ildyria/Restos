<script setup lang="ts">
import type { Course, RestaurantDetail } from '~~/shared/types'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ restaurant: RestaurantDetail, visitId: number | null, dishId: number | null }>()
const emit = defineEmits<{ updated: [detail: RestaurantDetail] }>()

const toast = useToast()
const currentId = ref<number | null>(null)
const state = reactive({ name: '', course: 'main' as Course | 'none', notes: '' })
const saving = ref(false)
const uploading = ref(false)
const files = ref<File[]>([])

const dish = computed(() => props.restaurant.visits.flatMap(v => v.dishes).find(d => d.id === currentId.value) ?? null)
const courseItems = [{ label: 'No course', value: 'none' }, ...COURSES]

watch(open, (isOpen) => {
  if (!isOpen) return
  currentId.value = props.dishId
  files.value = []
  const d = dish.value
  Object.assign(state, d ? { name: d.name, course: d.course ?? 'none', notes: d.notes } : { name: '', course: 'main', notes: '' })
})

function payload() {
  return { name: state.name, course: state.course === 'none' ? null : state.course, notes: state.notes }
}

async function save() {
  saving.value = true
  try {
    let detail: RestaurantDetail
    if (currentId.value === null) {
      detail = await api<RestaurantDetail>(`/api/admin/visits/${props.visitId}/dishes`, { method: 'POST', body: payload() })
      // Stay open in edit mode on the new dish so photos can be added right away.
      const ids = detail.visits.find(v => v.id === props.visitId)?.dishes.map(d => d.id) ?? []
      currentId.value = Math.max(...ids)
    } else {
      detail = await api<RestaurantDetail>(`/api/admin/dishes/${currentId.value}`, { method: 'PUT', body: payload() })
    }
    emit('updated', detail)
    toast.add({ title: 'Dish saved', color: 'success' })
  } catch (e) {
    toast.add({ title: 'Could not save dish', description: errorMessage(e), color: 'error' })
  } finally {
    saving.value = false
  }
}

watch(files, async (list) => {
  if (!list?.length || currentId.value === null) return
  uploading.value = true
  try {
    const fd = new FormData()
    for (const f of list) fd.append('files', f)
    emit('updated', await api<RestaurantDetail>(`/api/admin/dishes/${currentId.value}/photos`, { method: 'POST', body: fd }))
  } catch (e) {
    toast.add({ title: 'Upload failed', description: errorMessage(e), color: 'error' })
  } finally {
    files.value = []
    uploading.value = false
  }
})

async function setCover(photoId: number) {
  try {
    emit('updated', await api<RestaurantDetail>(`/api/admin/restaurants/${props.restaurant.id}/cover`, { method: 'PUT', body: { photo_id: photoId } }))
  } catch (e) {
    toast.add({ title: 'Could not set cover', description: errorMessage(e), color: 'error' })
  }
}

const photoToDelete = ref<number | null>(null)
const photoConfirm = computed({ get: () => photoToDelete.value !== null, set: (v) => { if (!v) photoToDelete.value = null } })

async function deletePhoto() {
  if (photoToDelete.value === null) return
  try {
    emit('updated', await api<RestaurantDetail>(`/api/admin/photos/${photoToDelete.value}`, { method: 'DELETE' }))
    photoToDelete.value = null
  } catch (e) {
    toast.add({ title: 'Could not delete photo', description: errorMessage(e), color: 'error' })
  }
}
</script>

<template>
  <USlideover v-model:open="open" :title="currentId === null ? 'Add a dish' : 'Edit dish'" description="What was ordered, and how it looked.">
    <template #body>
      <UForm :state="state" class="space-y-4" @submit="save">
        <UFormField label="Dish" name="name" required>
          <UInput v-model="state.name" required autofocus class="w-full" />
        </UFormField>
        <UFormField label="Course" name="course">
          <USelect v-model="state.course" :items="courseItems" class="w-full" />
        </UFormField>
        <UFormField label="Notes" name="notes">
          <UTextarea v-model="state.notes" :rows="3" autoresize class="w-full" />
        </UFormField>
        <UButton type="submit" :loading="saving">{{ currentId === null ? 'Add dish' : 'Save dish' }}</UButton>
      </UForm>

      <USeparator class="my-6" />

      <section>
        <h3 class="mb-3 text-sm font-medium">Photos</h3>
        <p v-if="currentId === null" class="text-sm text-muted">Save the dish first, then you can add photos.</p>
        <template v-else>
          <ul v-if="dish?.photos.length" class="mb-4 grid grid-cols-2 gap-3">
            <li v-for="p in dish.photos" :key="p.id" class="group relative overflow-hidden rounded-md bg-elevated">
              <img :src="mediaUrl(p.key)" :alt="dish.name" class="aspect-[4/3] w-full object-cover">
              <div class="absolute inset-x-0 bottom-0 flex justify-between gap-1 bg-gradient-to-t from-black/60 to-transparent p-1.5">
                <UButton
                  :icon="restaurant.cover_photo_id === p.id ? 'i-lucide-bookmark-check' : 'i-lucide-bookmark'"
                  size="xs"
                  :color="restaurant.cover_photo_id === p.id ? 'primary' : 'neutral'"
                  :variant="restaurant.cover_photo_id === p.id ? 'solid' : 'subtle'"
                  :aria-label="restaurant.cover_photo_id === p.id ? 'Current cover' : 'Use as cover'"
                  :title="restaurant.cover_photo_id === p.id ? 'Current cover' : 'Use as cover'"
                  @click="setCover(p.id)"
                />
                <UButton icon="i-lucide-trash-2" size="xs" color="error" variant="subtle" aria-label="Delete photo" @click="photoToDelete = p.id" />
              </div>
            </li>
          </ul>
          <UFileUpload
            v-model="files"
            multiple
            accept="image/jpeg,image/png,image/webp"
            label="Drop photos here or click to browse"
            description="JPEG, PNG or WebP, up to 20 MB each"
            :disabled="uploading"
            class="min-h-32"
          />
          <p v-if="uploading" class="mt-2 text-sm text-muted">Uploading…</p>
        </template>
      </section>

      <ConfirmModal v-model:open="photoConfirm" title="Delete photo?" description="This can't be undone." @confirm="deletePhoto" />
    </template>
  </USlideover>
</template>
