<script setup lang="ts">
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

export interface MapPin {
  id: number
  label: string
  name: string
  lat: number
  lng: number
}

const props = defineProps<{ pins: MapPin[], active?: number | null }>()
const emit = defineEmits<{ hover: [id: number | null], select: [id: number] }>()

const el = ref<HTMLElement>()
let map: L.Map | null = null
let markers = new Map<number, L.Marker>()
let resize: ResizeObserver | null = null

function pinIcon(p: MapPin, on: boolean) {
  const size = on ? 30 : 24
  return L.divIcon({
    className: '',
    html: `<span class="map-pin${on ? ' is-active' : ''}">${p.label}</span>`,
    iconSize: [size, size],
    iconAnchor: [size / 2, size / 2]
  })
}

function drawPins() {
  if (!map) return
  for (const m of markers.values()) m.remove()
  markers = new Map()
  for (const p of props.pins) {
    const m = L.marker([p.lat, p.lng], { icon: pinIcon(p, p.id === props.active), title: p.name, keyboard: true, riseOnHover: true })
      .on('mouseover', () => emit('hover', p.id))
      .on('mouseout', () => emit('hover', null))
      .on('click', () => emit('select', p.id))
      .addTo(map)
    markers.set(p.id, m)
  }
  if (props.pins.length) {
    map.fitBounds(L.latLngBounds(props.pins.map(p => [p.lat, p.lng] as L.LatLngTuple)), { padding: [40, 40], maxZoom: 15 })
  }
}

onMounted(() => {
  map = L.map(el.value!, { zoomControl: true, scrollWheelZoom: false }).setView([50.85, 4.35], 8)
  baseTiles().addTo(map)
  drawPins()
  // The map can be mounted while its box is still settling (sticky column, tab switch).
  resize = new ResizeObserver(() => map?.invalidateSize())
  resize.observe(el.value!)
})

onBeforeUnmount(() => {
  resize?.disconnect()
  map?.remove()
  map = null
})

watch(() => props.pins.map(p => `${p.id}:${p.label}:${p.lat}:${p.lng}`).join('|'), drawPins)
watch(() => props.active, (cur, prev) => {
  for (const id of [prev, cur]) {
    const p = props.pins.find(x => x.id === id)
    const m = id != null ? markers.get(id) : undefined
    if (!p || !m) continue
    m.setIcon(pinIcon(p, id === cur))
    m.setZIndexOffset(id === cur ? 1000 : 0)
  }
})
</script>

<template>
  <div ref="el" class="size-full" />
</template>

