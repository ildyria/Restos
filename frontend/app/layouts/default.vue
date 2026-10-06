<script setup lang="ts">
const { user } = useAuth()
const { refresh, checked } = useAuth()
onMounted(() => { if (!checked.value) refresh() })

const { names } = usePeople()
const nav = [{ to: '/', label: 'Ranking' }, { to: '/about', label: 'About us' }]
const route = useRoute()
// Restaurant pages belong to the ranking section.
const isActive = (to: string) => to === '/' ? route.path === '/' || route.path.startsWith('/r/') : route.path === to

const today = new Date().toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })
</script>

<template>
  <div class="journal flex min-h-screen flex-col">
    <header class="mx-auto w-full max-w-[1280px] px-4 sm:px-10">
      <div class="grid grid-cols-1 items-center gap-1 pb-3.5 pt-4 text-center sm:grid-cols-[1fr_auto_1fr] sm:pt-[18px]">
        <div class="hidden text-xs sm:block">{{ today }}</div>
        <NuxtLink to="/" class="font-display text-4xl leading-none sm:text-5xl">{{ SITE_NAME }}</NuxtLink>
        <div class="hidden items-center justify-end gap-2 text-xs text-muted sm:flex">
          <span v-if="names">Restaurants reviewed by {{ names }}</span>
          <UButton v-if="user" to="/admin" color="neutral" variant="ghost" size="xs" icon="i-lucide-pencil-line">Admin</UButton>
          <UColorModeButton color="neutral" variant="ghost" size="xs" />
        </div>
      </div>
      <nav class="rule-double flex justify-center gap-7 py-2 text-sm sm:py-[9px]" aria-label="Sections">
        <NuxtLink
          v-for="n in nav"
          :key="n.to"
          :to="n.to"
          class="underline-offset-[7px] hover:text-highlighted"
          :class="isActive(n.to) ? 'text-highlighted underline decoration-1' : 'text-muted'"
          :aria-current="isActive(n.to) ? 'page' : undefined"
        >
          {{ n.label }}
        </NuxtLink>
      </nav>
    </header>

    <main class="mx-auto w-full max-w-[1280px] flex-1 px-4 py-10 sm:px-10 sm:py-11">
      <slot />
    </main>

    <footer class="border-t border-default">
      <div class="mx-auto flex max-w-[1280px] items-center justify-between px-4 py-6 text-xs text-dimmed sm:px-10">
        <span>Places we've eaten, in order of preference.</span>
        <NuxtLink v-if="!user" to="/admin/login" class="hover:text-default">Sign in</NuxtLink>
      </div>
    </footer>
  </div>
</template>
