<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'

const { user, logout } = useAuth()
const toast = useToast()

const items: NavigationMenuItem[][] = [[
  { label: 'Restaurants', icon: 'i-lucide-utensils', to: '/admin', exact: true },
  { label: 'New restaurant', icon: 'i-lucide-plus', to: '/admin/restaurants/new' },
  { label: 'Ranking', icon: 'i-lucide-list-ordered', to: '/admin/ranking' },
  { label: 'Admins', icon: 'i-lucide-users', to: '/admin/users' }
], [
  { label: 'View site', icon: 'i-lucide-external-link', to: '/' }
]]

async function signOut() {
  try {
    await logout()
    await navigateTo('/admin/login')
  } catch (e) {
    toast.add({ title: 'Could not sign out', description: errorMessage(e), color: 'error' })
  }
}
</script>

<template>
  <UDashboardGroup>
    <UDashboardSidebar collapsible resizable :ui="{ footer: 'border-t border-default' }">
      <template #header>
        <NuxtLink to="/admin" class="font-display px-2 text-xl font-medium">Restos <span class="text-sm text-dimmed">admin</span></NuxtLink>
      </template>

      <template #default="{ collapsed }">
        <UNavigationMenu :collapsed="collapsed" :items="items[0]" orientation="vertical" />
        <UNavigationMenu :collapsed="collapsed" :items="items[1]" orientation="vertical" class="mt-auto" />
      </template>

      <template #footer="{ collapsed }">
        <div class="flex w-full items-center justify-between gap-2">
          <span v-if="!collapsed" class="truncate text-sm text-muted">{{ user?.name }}</span>
          <div class="flex">
            <UColorModeButton color="neutral" variant="ghost" size="sm" />
            <UButton icon="i-lucide-log-out" color="neutral" variant="ghost" size="sm" aria-label="Sign out" @click="signOut" />
          </div>
        </div>
      </template>
    </UDashboardSidebar>

    <slot />
  </UDashboardGroup>
</template>
