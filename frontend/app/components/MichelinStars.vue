<script setup lang="ts">
// Rosette approximating the Michelin Guide star: six rounded petals around a hollow centre.
defineProps<{ count: number }>()
const petals = Array.from({ length: 6 }, (_, i) => {
  const a = (i * Math.PI) / 3
  return { x: 12 + 5.2 * Math.sin(a), y: 12 - 5.2 * Math.cos(a) }
})
</script>

<template>
  <span class="inline-flex items-center gap-0.5 align-middle text-red-600" :title="`${count} Michelin star${count > 1 ? 's' : ''}`">
    <svg v-for="n in count" :key="n" viewBox="0 0 24 24" class="size-[1.1em]" aria-hidden="true">
      <mask id="michelin-hole">
        <rect width="24" height="24" fill="#fff" />
        <circle cx="12" cy="12" r="2" fill="#000" />
      </mask>
      <g fill="currentColor" mask="url(#michelin-hole)">
        <circle v-for="(p, i) in petals" :key="i" :cx="p.x" :cy="p.y" r="4.6" />
        <circle cx="12" cy="12" r="5" />
      </g>
    </svg>
    <span class="sr-only">{{ count }} Michelin star{{ count > 1 ? 's' : '' }}</span>
  </span>
</template>
