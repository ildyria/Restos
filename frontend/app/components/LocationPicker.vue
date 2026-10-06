<script setup lang="ts">
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

// Click the map or drag the pin to place the restaurant; "Find address" geocodes via OpenStreetMap.
const lat = defineModel<number | null>('lat', { required: true })
const lng = defineModel<number | null>('lng', { required: true })
const props = defineProps<{ address: string, city: string }>()

const toast = useToast()
const el = ref<HTMLElement>()
let map: L.Map | null = null
let marker: L.Marker | null = null

const icon = L.divIcon({ className: '', html: '<span class="map-pin is-active"></span>', iconSize: [24, 24], iconAnchor: [12, 12] })
const round = (n: number) => Math.round(n * 1e6) / 1e6

function place(la: number, ln: number) {
  lat.value = round(la)
  lng.value = round(ln)
}

function sync() {
  if (!map) return
  if (lat.value == null || lng.value == null) {
    marker?.remove()
    marker = null
    return
  }
  const pos: L.LatLngTuple = [lat.value, lng.value]
  if (marker) {
    marker.setLatLng(pos)
  } else {
    marker = L.marker(pos, { icon, draggable: true }).addTo(map)
    marker.on('dragend', () => { const p = marker!.getLatLng(); place(p.lat, p.lng) })
  }
}

onMounted(() => {
  const start: L.LatLngTuple = lat.value != null && lng.value != null ? [lat.value, lng.value] : [50.85, 4.35]
  map = L.map(el.value!, { scrollWheelZoom: false }).setView(start, lat.value != null ? 16 : 7)
  baseTiles().addTo(map)
  map.on('click', e => place(e.latlng.lat, e.latlng.lng))
  sync()
})
onBeforeUnmount(() => { map?.remove(); map = null; marker = null })
watch([lat, lng], sync)

const searching = ref(false)
const note = ref('')
const query = computed(() => [props.address, props.city].map(s => s.trim()).filter(Boolean).join(', '))
// An existing pin already matches the address it was saved with; don't look that up again.
let lastLooked = lat.value != null ? query.value : ''

// Called when the address or city field is left: places the pin from the address. Needs a street
// address (a city alone would drop the pin in the city centre) and skips unchanged addresses.
function autoLookup() {
  if (!props.address.trim() || query.value === lastLooked) return
  return lookup(true)
}
defineExpose({ autoLookup })

async function lookup(auto = false) {
  if (!query.value) return
  lastLooked = query.value
  searching.value = true
  note.value = ''
  try {
    const res = await $fetch<{ lat: string, lon: string }[]>('https://nominatim.openstreetmap.org/search', {
      query: { format: 'json', limit: 1, q: query.value },
      referrerPolicy: OSM_REFERRER_POLICY
    })
    const hit = res[0]
    if (!hit) {
      if (auto) note.value = 'Address not found on the map. Click the map to place the pin.'
      else toast.add({ title: 'Address not found', description: 'Click the map to place the pin instead.', color: 'warning' })
      return
    }
    const la = Number(hit.lat), ln = Number(hit.lon)
    place(la, ln)
    // Not lat.value: the model only reflects the new value once the parent has re-rendered.
    map?.setView([la, ln], 16)
    if (auto) note.value = 'Placed from the address. Drag the pin if it’s off.'
  } catch {
    if (auto) note.value = 'Couldn’t look up the address. Click the map to place the pin.'
    else toast.add({ title: 'Lookup failed', description: 'Click the map to place the pin instead.', color: 'error' })
  } finally {
    searching.value = false
  }
}
</script>

<template>
  <div>
    <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
      <p class="text-sm text-muted">
        <template v-if="searching">Looking up the address…</template>
        <template v-else-if="note">{{ note }}</template>
        <template v-else-if="lat != null && lng != null">{{ lat.toFixed(5) }}, {{ lng.toFixed(5) }}</template>
        <template v-else>Not on the map yet. Enter the address, or click the map.</template>
      </p>
      <div class="flex gap-1">
        <UButton size="xs" color="neutral" variant="subtle" icon="i-lucide-search" :loading="searching" :disabled="!query" @click="lookup()">
          Find address
        </UButton>
        <UButton v-if="lat != null" size="xs" color="neutral" variant="ghost" icon="i-lucide-x" @click="lat = null; lng = null">
          Remove
        </UButton>
      </div>
    </div>
    <div ref="el" class="h-72 overflow-hidden rounded-lg border border-default" />
  </div>
</template>
