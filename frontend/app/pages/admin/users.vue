<script setup lang="ts">
import type { User } from '~~/shared/types'

definePageMeta({ layout: 'admin', middleware: 'admin' })
useSeoMeta({ title: `Admins — ${SITE_NAME} admin`, robots: 'noindex' })

const toast = useToast()
const { user: me } = useAuth()
const { data: users, refresh } = await useFetch<User[]>('/api/admin/users', { default: () => [] })

const state = reactive({ name: '', email: '', password: '' })
const saving = ref(false)

async function add() {
  saving.value = true
  try {
    await api('/api/admin/users', { method: 'POST', body: { ...state } })
    toast.add({ title: `Added ${state.name}`, color: 'success' })
    Object.assign(state, { name: '', email: '', password: '' })
    await refresh()
  } catch (e) {
    toast.add({ title: 'Could not add admin', description: errorMessage(e), color: 'error' })
  } finally {
    saving.value = false
  }
}

const toDelete = ref<User | null>(null)
const confirmOpen = computed({ get: () => !!toDelete.value, set: (v) => { if (!v) toDelete.value = null } })

async function remove() {
  if (!toDelete.value) return
  try {
    await api(`/api/admin/users/${toDelete.value.id}`, { method: 'DELETE' })
    toast.add({ title: `Removed ${toDelete.value.name}`, color: 'success' })
    toDelete.value = null
    await refresh()
  } catch (e) {
    toast.add({ title: 'Could not remove admin', description: errorMessage(e), color: 'error' })
  }
}
</script>

<template>
  <UDashboardPanel>
    <template #header>
      <UDashboardNavbar title="Admins" />
    </template>

    <template #body>
      <div class="grid max-w-3xl gap-10">
        <ul class="divide-y divide-default rounded-lg border border-default">
          <li v-for="u in users" :key="u.id" class="flex items-center justify-between px-4 py-3">
            <div>
              <p class="font-medium">{{ u.name }} <span v-if="u.id === me?.id" class="text-xs text-dimmed">(you)</span></p>
              <p class="text-sm text-muted">{{ u.email }}</p>
            </div>
            <UButton
              v-if="u.id !== me?.id"
              icon="i-lucide-trash-2"
              color="error"
              variant="ghost"
              size="sm"
              :aria-label="`Remove ${u.name}`"
              @click="toDelete = u"
            />
          </li>
        </ul>

        <section>
          <h2 class="font-display mb-3 text-xl">Add an admin</h2>
          <UForm :state="state" class="grid gap-4 sm:grid-cols-2" @submit="add">
            <UFormField label="Name" name="name"><UInput v-model="state.name" required class="w-full" /></UFormField>
            <UFormField label="Email" name="email"><UInput v-model="state.email" type="email" required class="w-full" /></UFormField>
            <UFormField label="Password" name="password" help="At least 8 characters" class="sm:col-span-2">
              <UInput v-model="state.password" type="password" autocomplete="new-password" required minlength="8" class="w-full" />
            </UFormField>
            <div class="sm:col-span-2"><UButton type="submit" :loading="saving">Add admin</UButton></div>
          </UForm>
        </section>
      </div>

      <UModal v-model:open="confirmOpen" title="Remove admin?" :description="`${toDelete?.name} will no longer be able to sign in.`">
        <template #footer>
          <div class="flex w-full justify-end gap-2">
            <UButton color="neutral" variant="ghost" @click="toDelete = null">Cancel</UButton>
            <UButton color="error" @click="remove">Remove</UButton>
          </div>
        </template>
      </UModal>
    </template>
  </UDashboardPanel>
</template>
