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

// active: the highlighted pin. focus: the pin to zoom to (null: show every pin).
const props = defineProps<{ pins: MapPin[], active?: number | null, focus?: number | null }>()
const emit = defineEmits<{ hover: [id: number | null], select: [id: number] }>()

const el = ref<HTMLElement>()
let map: L.Map | null = null
let markers = new Map<number, L.Marker>()
let resize: ResizeObserver | null = null
let focusTimer: ReturnType<typeof setTimeout> | undefined
const FOCUS_ZOOM = 15

// The view to return to once nothing is focused: whatever the visitor was looking at (their own
// zoom included) before the map flew to a hovered restaurant.
interface View { center: L.LatLng, zoom: number }
let home: View | null = null
// Set while flying back home, so a restaurant hovered mid-flight still returns to the same view
// rather than to wherever the flight happened to be.
let returningTo: View | null = null

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
  home = returningTo = null
  showAll()
}

const reducedMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches

function showAll() {
  if (!map || !props.pins.length) return
  map.fitBounds(L.latLngBounds(props.pins.map(p => [p.lat, p.lng] as L.LatLngTuple)), { padding: [40, 40], maxZoom: FOCUS_ZOOM })
}

function moveTo(center: L.LatLngExpression, zoom: number) {
  if (!map) return
  if (reducedMotion()) map.setView(center, zoom, { animate: false })
  else map.flyTo(center, zoom, { duration: 0.6 })
}

function zoomTo(p: MapPin) {
  if (!map) return
  home ??= returningTo ?? { center: map.getCenter(), zoom: map.getZoom() }
  moveTo([p.lat, p.lng], Math.max(map.getZoom(), FOCUS_ZOOM))
}

function goHome() {
  if (!map || !home) return
  const view = home
  home = null
  moveTo(view.center, view.zoom)
  returningTo = view
}

onMounted(() => {
  map = L.map(el.value!, { zoomControl: true, scrollWheelZoom: true }).setView([50.85, 4.35], 8)
  baseTiles().addTo(map)
  // Any finished move (the flight home, or the visitor's own zoom or pan) ends the return trip.
  map.on('moveend', () => { returningTo = null })
  drawPins()
  // The map can be mounted while its box is still settling (sticky column, tab switch).
  resize = new ResizeObserver(() => map?.invalidateSize())
  resize.observe(el.value!)
})

onBeforeUnmount(() => {
  clearTimeout(focusTimer)
  resize?.disconnect()
  map?.remove()
  map = null
})

watch(() => props.pins.map(p => `${p.id}:${p.label}:${p.lat}:${p.lng}`).join('|'), drawPins)
// Wait a moment before moving, so sweeping the mouse across the list doesn't make the map jump
// from pin to pin, and a short gap between two rows doesn't zoom out in between.
watch(() => props.focus, (id) => {
  clearTimeout(focusTimer)
  focusTimer = setTimeout(() => {
    const p = id != null ? props.pins.find(x => x.id === id) : undefined
    if (p) zoomTo(p)
    else goHome()
  }, id != null ? 200 : 500)
})

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

