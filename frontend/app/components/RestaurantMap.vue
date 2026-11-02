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
// mapClick: a click on the map itself (not on a pin, the controls, or the end of a drag).
const emit = defineEmits<{ hover: [id: number | null], select: [id: number], mapClick: [] }>()

const el = ref<HTMLElement>()
let map: L.Map | null = null
let markers = new Map<number, L.Marker>()
let resize: ResizeObserver | null = null
let focusTimer: ReturnType<typeof setTimeout> | undefined
const FOCUS_ZOOM = 15

// The view to return to once nothing is focused: whatever the visitor was looking at (their own
// zoom included) before the map flew to a hovered restaurant. 'overview' is the fitted view of
// every pin, worked out again for the map's size when it's used (the map may have been resized).
interface View { center: L.LatLng, zoom: number }
type Home = View | 'overview'
let home: Home | null = null
// Set while flying back home, so a restaurant hovered mid-flight still returns to the same view
// rather than to wherever the flight happened to be.
let returningTo: Home | null = null

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

const pinBounds = () => L.latLngBounds(props.pins.map(p => [p.lat, p.lng] as L.LatLngTuple))

function showAll() {
  if (!map || !props.pins.length) return
  map.fitBounds(pinBounds(), { padding: [40, 40], maxZoom: FOCUS_ZOOM, animate: false })
}

// The view showAll() gives at the map's current size (the same maths as Leaflet's fitBounds).
function overviewView(): View | null {
  if (!map || !props.pins.length) return null
  const b = pinBounds()
  const zoom = Math.min(FOCUS_ZOOM, map.getBoundsZoom(b, false, L.point(80, 80)))
  const center = map.unproject(map.project(b.getSouthWest(), zoom).add(map.project(b.getNorthEast(), zoom)).divideBy(2), zoom)
  return { center, zoom }
}

// Whether the map still shows that overview, i.e. the visitor hasn't zoomed or moved it.
function onOverview() {
  const o = overviewView()
  if (!map || !o || map.getZoom() !== o.zoom) return false
  return map.project(map.getCenter()).distanceTo(map.project(o.center)) < 2
}

// After a size change: an untouched overview is fitted again to the new size; a view the visitor
// chose themselves is kept.
function refresh() {
  if (!map) return
  const fit = onOverview()
  map.invalidateSize()
  if (fit) showAll()
}

// While the page scrolls (index.vue bringing a hovered pin's row into view), keeps the map's
// picture where it is on the screen: when the scroll moves the map's box, its view moves back by as
// much, so the pin stays under the mouse. Ends once the page has been still for a moment.
let holding: (() => void) | null = null
function holdInPlace() {
  holding?.()
  const box = el.value
  if (!map || !box) return
  let top = box.getBoundingClientRect().top
  let idle: ReturnType<typeof setTimeout>
  const onScroll = () => {
    const dy = Math.round(box.getBoundingClientRect().top - top)
    if (dy && map) map.panBy([0, dy], { animate: false })
    top += dy
    clearTimeout(idle)
    idle = setTimeout(stop, 200)
  }
  const stop = () => {
    clearTimeout(idle)
    window.removeEventListener('scroll', onScroll)
    holding = null
  }
  window.addEventListener('scroll', onScroll, { passive: true })
  idle = setTimeout(stop, 200)
  holding = stop
}
defineExpose({ refresh, holdInPlace })

// Moves are unhurried. A place near the current one at the same zoom (the next restaurant down the
// street) is reached by a glide; a flight would zoom out and back in on the way.
function moveTo(center: L.LatLngExpression, zoom: number) {
  if (!map) return
  const size = map.getSize()
  const near = zoom === map.getZoom() && map.project(center).distanceTo(map.project(map.getCenter())) < 2 * Math.max(size.x, size.y)
  if (reducedMotion()) map.setView(center, zoom, { animate: false })
  else if (near) map.panTo(center, { animate: true, duration: 0.9, easeLinearity: 0.5 })
  else map.flyTo(center, zoom, { duration: 1.2 })
}

function zoomTo(p: MapPin) {
  if (!map) return
  home ??= returningTo ?? (onOverview() ? 'overview' : { center: map.getCenter(), zoom: map.getZoom() })
  moveTo([p.lat, p.lng], Math.max(map.getZoom(), FOCUS_ZOOM))
}

function goHome() {
  if (!map || !home) return
  const back = home
  const view = back === 'overview' ? overviewView() : back
  home = null
  if (view) moveTo(view.center, view.zoom)
  returningTo = back
}

onMounted(() => {
  map = L.map(el.value!, { zoomControl: true, scrollWheelZoom: true }).setView([50.85, 4.35], 8)
  baseTiles().addTo(map)
  // Any finished move (the flight home, or the visitor's own zoom or pan) ends the return trip.
  map.on('moveend', () => { returningTo = null })
  map.on('click', () => emit('mapClick'))
  drawPins()
  // The map can be mounted while its box is still settling (sticky column, tab switch).
  resize = new ResizeObserver(refresh)
  resize.observe(el.value!)
})

onBeforeUnmount(() => {
  clearTimeout(focusTimer)
  holding?.()
  resize?.disconnect()
  map?.remove()
  map = null
})

watch(() => props.pins.map(p => `${p.id}:${p.label}:${p.lat}:${p.lng}`).join('|'), drawPins)
// Wait for the mouse to settle on a row before moving, so sweeping across the list doesn't make
// the map jump from pin to pin, and a short gap between two rows doesn't zoom out in between.
watch(() => props.focus, (id) => {
  clearTimeout(focusTimer)
  focusTimer = setTimeout(() => {
    const p = id != null ? props.pins.find(x => x.id === id) : undefined
    if (p) zoomTo(p)
    else goHome()
  }, id != null ? 350 : 500)
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

