<script setup lang="ts">
import type { Photo, RestaurantDetail } from '~~/shared/types'

const route = useRoute()
const { data: r, error } = await useFetch<RestaurantDetail>(`/api/restaurants/${route.params.slug}`)

useSeoMeta({ title: () => (r.value ? `${r.value.name} — ${SITE_NAME}` : SITE_NAME), description: () => r.value?.review.slice(0, 160) })

// One flat gallery so the lightbox can step through every dish photo of this restaurant.
const gallery = computed(() => {
  const items: { photo: Photo, caption: string }[] = []
  for (const v of r.value?.visits ?? []) {
    for (const d of v.dishes) {
      for (const p of d.photos) items.push({ photo: p, caption: `${d.name} · ${formatDate(v.visited_on)}` })
    }
    for (const p of v.photos) items.push({ photo: p, caption: formatDate(v.visited_on) })
  }
  return items
})
const photos = computed(() => gallery.value.map(g => g.photo))
const lightbox = ref<number | null>(null)
const captionFor = (p: Photo) => gallery.value.find(g => g.photo.id === p.id)?.caption ?? ''
const openPhoto = (p: Photo) => { lightbox.value = photos.value.findIndex(x => x.id === p.id) }

const courseLabel = (c: string | null) => COURSES.find(x => x.value === c)?.label
</script>

<template>
  <div v-if="error" class="py-20 text-center">
    <p class="font-display text-3xl">We haven't been there.</p>
    <UButton to="/" class="mt-6" color="neutral" variant="subtle">Back to the list</UButton>
  </div>

  <article v-else-if="r">
    <div class="mb-6 flex items-center justify-between gap-4">
      <NuxtLink to="/" class="inline-flex items-center gap-1 text-sm text-muted hover:text-default">
        <UIcon name="i-lucide-arrow-left" class="size-4" /> All restaurants
      </NuxtLink>
      <AdminEditButton :id="r.id" :name="r.name" variant="subtle" size="sm">Edit</AdminEditButton>
    </div>

    <div v-if="r.cover_key" class="mb-8 overflow-hidden rounded-lg bg-elevated">
      <img :src="mediaUrl(r.cover_key, 'full')" :alt="r.name" class="max-h-[28rem] w-full object-cover">
    </div>

    <header class="grid gap-8 md:grid-cols-[1fr_20rem]">
      <div>
        <p v-if="r.rank_position" class="rank-numeral text-lg text-primary">
          № {{ r.rank_position }} on our list
        </p>
        <h1 class="font-display text-4xl font-medium leading-tight sm:text-5xl">{{ r.name }}<MichelinStars v-if="r.michelin_stars" :count="r.michelin_stars" class="ml-3 text-2xl" /></h1>
        <p v-if="r.cuisine || r.price_level" class="mt-3 text-muted">
          <span v-if="r.cuisine">{{ r.cuisine }}</span>
          <span v-if="r.cuisine && r.price_level"> · </span>
          <span v-if="r.price_level">{{ priceLabel(r.price_level) }}</span>
        </p>
        <p v-if="r.address || r.city" class="mt-1 flex items-center gap-1.5 text-sm text-muted">
          <UIcon name="i-lucide-map-pin" class="size-4 shrink-0" />
          {{ [r.address, r.city].filter(Boolean).join(', ') }}
        </p>
        <p v-if="r.website" class="mt-1 text-sm">
          <a :href="r.website" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1.5 text-primary hover:underline">
            <UIcon name="i-lucide-external-link" class="size-4" /> Website
          </a>
        </p>
      </div>

      <div class="rounded-lg border border-default p-5">
        <div class="mb-4 flex items-baseline justify-between">
          <span class="font-display text-5xl tabular-nums">{{ formatScore(r.overall) }}</span>
          <RatingStars :value="r.overall" :size="18" />
        </div>
        <ScoreBreakdown :restaurant="r" />
      </div>
    </header>

    <section v-if="r.review" class="mt-10 max-w-2xl">
      <p class="font-display whitespace-pre-line text-xl leading-relaxed">{{ r.review }}</p>
    </section>

    <section class="mt-14">
      <h2 class="font-display text-2xl">Visits</h2>
      <p v-if="!r.visits.length" class="mt-3 text-sm text-muted">We don't remember when we went.</p>

      <ol v-else class="mt-6 space-y-10 border-l border-default pl-6">
        <li v-for="v in r.visits" :key="v.id" class="relative">
          <span class="absolute -left-[1.85rem] top-2 size-2.5 rounded-full bg-primary ring-4 ring-default" />
          <div class="flex flex-wrap items-center gap-2">
            <h3 class="font-medium">{{ formatDate(v.visited_on) }}</h3>
            <UBadge :label="v.meal === 'lunch' ? 'Lunch' : 'Dinner'" color="neutral" variant="subtle" size="sm" />
          </div>
          <p v-if="v.notes" class="mt-2 max-w-2xl text-muted">{{ v.notes }}</p>

          <ul v-if="v.dishes.length" class="mt-5 grid gap-6 sm:grid-cols-2">
            <li v-for="d in v.dishes" :key="d.id">
              <div v-if="d.photos.length" class="mb-2 grid gap-1.5" :class="d.photos.length > 1 ? 'grid-cols-2' : 'grid-cols-1'">
                <button
                  v-for="p in d.photos"
                  :key="p.id"
                  type="button"
                  class="aspect-[4/3] cursor-zoom-in overflow-hidden rounded-md bg-elevated"
                  :aria-label="`View photo of ${d.name}`"
                  @click="openPhoto(p)"
                >
                  <img :src="mediaUrl(p.key)" :alt="d.name" class="size-full object-cover transition-transform duration-500 hover:scale-[1.03]" loading="lazy">
                </button>
              </div>
              <p class="font-medium leading-snug">{{ d.name }}</p>
              <p v-if="d.course" class="text-xs uppercase tracking-wide text-dimmed">{{ courseLabel(d.course) }}</p>
              <p v-if="d.notes" class="mt-1 text-sm text-muted">{{ d.notes }}</p>
            </li>
          </ul>

          <div v-if="v.photos.length" class="mt-5 grid grid-cols-2 gap-1.5 sm:grid-cols-4">
            <button
              v-for="p in v.photos"
              :key="p.id"
              type="button"
              class="aspect-[4/3] cursor-zoom-in overflow-hidden rounded-md bg-elevated"
              :aria-label="`View photo from ${formatDate(v.visited_on)}`"
              @click="openPhoto(p)"
            >
              <img :src="mediaUrl(p.key)" :alt="`Photo from ${formatDate(v.visited_on)}`" class="size-full object-cover transition-transform duration-500 hover:scale-[1.03]" loading="lazy">
            </button>
          </div>
        </li>
      </ol>
    </section>

    <PhotoLightbox v-model="lightbox" :photos="photos" :caption="captionFor" />
  </article>
</template>
