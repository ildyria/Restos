<script setup lang="ts">
const { user } = useAuth()
const { refresh, checked } = useAuth()
onMounted(() => { if (!checked.value) refresh() })
</script>

<template>
  <div class="flex min-h-screen flex-col">
    <header class="border-b border-default">
      <div class="mx-auto flex h-16 max-w-5xl items-center justify-between px-4 sm:px-6">
        <NuxtLink to="/" class="font-display text-2xl font-medium tracking-tight">
          Restos
        </NuxtLink>
        <nav class="flex items-center gap-1">
          <UButton v-if="user" to="/admin" color="neutral" variant="ghost" size="sm" icon="i-lucide-pencil-line">
            Admin
          </UButton>
          <UColorModeButton color="neutral" variant="ghost" size="sm" />
        </nav>
      </div>
    </header>

    <main class="mx-auto w-full max-w-5xl flex-1 px-4 py-10 sm:px-6">
      <slot />
    </main>

    <footer class="border-t border-default">
      <div class="mx-auto flex max-w-5xl items-center justify-between px-4 py-6 text-xs text-dimmed sm:px-6">
        <span>Places we've eaten, in order of preference.</span>
        <NuxtLink v-if="!user" to="/admin/login" class="hover:text-default">Sign in</NuxtLink>
      </div>
    </footer>
  </div>
</template>
