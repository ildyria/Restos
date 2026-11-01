<script setup lang="ts">
// Shortcut from the public pages to a restaurant's admin page, shown only to logged-in admins.
// Clicks and keys stop here so a surrounding clickable row doesn't also react.
defineProps<{ id: number, name: string }>()
const { user } = useAuth()
</script>

<template>
  <UButton
    v-if="user"
    :to="`/admin/restaurants/${id}`"
    icon="i-lucide-pencil"
    size="xs"
    color="neutral"
    variant="ghost"
    :aria-label="`Edit ${name}`"
    :title="`Edit ${name}`"
    @click.stop
    @keydown.stop
  >
    <slot />
  </UButton>
</template>
