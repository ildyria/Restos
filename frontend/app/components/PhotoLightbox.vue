<script setup lang="ts">
import type { Photo } from '~~/shared/types'

const props = defineProps<{ photos: Photo[], caption?: (p: Photo) => string }>()
const index = defineModel<number | null>({ default: null })

const open = computed({
  get: () => index.value !== null,
  set: (v) => { if (!v) index.value = null }
})
const current = computed(() => (index.value === null ? undefined : props.photos[index.value]))

function step(d: number) {
  if (index.value === null) return
  index.value = (index.value + d + props.photos.length) % props.photos.length
}

function onKey(e: KeyboardEvent) {
  if (!open.value) return
  if (e.key === 'ArrowLeft') step(-1)
  if (e.key === 'ArrowRight') step(1)
}
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <UModal
    v-model:open="open"
    title="Photo"
    description="Dish photo viewer"
    :ui="{ content: 'max-w-5xl bg-transparent ring-0 shadow-none', header: 'sr-only', body: 'p-0 sm:p-0' }"
  >
    <template #body>
      <div v-if="current" class="relative">
        <img
          :src="mediaUrl(current.key, 'full')"
          :alt="caption?.(current) ?? ''"
          class="mx-auto max-h-[80vh] w-auto rounded-md object-contain"
        >
        <p v-if="caption" class="mt-3 text-center text-sm text-white/90">{{ caption(current) }}</p>
        <template v-if="photos.length > 1">
          <UButton
            icon="i-lucide-chevron-left"
            color="neutral"
            variant="solid"
            class="absolute left-2 top-1/2 -translate-y-1/2 rounded-full"
            aria-label="Previous photo"
            @click="step(-1)"
          />
          <UButton
            icon="i-lucide-chevron-right"
            color="neutral"
            variant="solid"
            class="absolute right-2 top-1/2 -translate-y-1/2 rounded-full"
            aria-label="Next photo"
            @click="step(1)"
          />
        </template>
      </div>
    </template>
  </UModal>
</template>
