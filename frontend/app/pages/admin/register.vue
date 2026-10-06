<script setup lang="ts">
import { z } from 'zod'

definePageMeta({ layout: false })
useSeoMeta({ title: `Create an account — ${SITE_NAME}`, robots: 'noindex' })

const toast = useToast()
const { register, user, checked, refresh } = useAuth()
const { data: authConfig, status } = useAuthConfig()

const state = reactive({ name: '', email: '', password: '', confirm: '' })
const schema = z.object({
  name: z.string().trim().min(1, 'Name is required').max(100),
  email: z.string().trim().email('Enter a valid email address'),
  password: z.string().min(8, 'At least 8 characters').max(72, 'At most 72 characters'),
  confirm: z.string()
}).refine(s => s.password === s.confirm, { path: ['confirm'], message: 'Passwords don’t match' })
const loading = ref(false)

onMounted(async () => {
  if (!checked.value) await refresh()
  if (user.value) await navigateTo('/admin')
})

async function submit() {
  loading.value = true
  try {
    // The confirmation is checked here only; the server just gets the password.
    await register({ name: state.name, email: state.email, password: state.password })
    toast.add({ title: `Welcome, ${user.value?.name}`, color: 'success' })
    await navigateTo('/admin')
  } catch (e) {
    toast.add({ title: 'Could not create the account', description: errorMessage(e), color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UApp>
    <div class="grid min-h-screen place-items-center px-4">
      <div class="w-full max-w-sm">
        <NuxtLink to="/" class="font-display mb-8 block text-center text-3xl font-medium">{{ SITE_NAME }}</NuxtLink>

        <div v-if="status === 'pending'" class="text-center text-sm text-dimmed">Loading…</div>

        <div v-else-if="!authConfig?.registration" class="rounded-lg border border-default p-6 text-center">
          <p class="font-display text-xl">Registration is closed</p>
          <p class="mt-2 text-sm text-muted">Ask an existing admin to add you.</p>
          <UButton to="/admin/login" class="mt-5" color="neutral" variant="subtle">Sign in</UButton>
        </div>

        <template v-else>
          <UForm :state="state" :schema="schema" class="space-y-4 rounded-lg border border-default p-6" @submit="submit">
            <UFormField label="Name" name="name" help="Shown on the site next to your scores.">
              <UInput v-model="state.name" autocomplete="name" required autofocus class="w-full" />
            </UFormField>
            <UFormField label="Email" name="email">
              <UInput v-model="state.email" type="email" autocomplete="username" required class="w-full" />
            </UFormField>
            <UFormField label="Password" name="password" help="At least 8 characters.">
              <UInput v-model="state.password" type="password" autocomplete="new-password" required class="w-full" />
            </UFormField>
            <UFormField label="Confirm password" name="confirm">
              <UInput v-model="state.confirm" type="password" autocomplete="new-password" required class="w-full" />
            </UFormField>
            <UButton type="submit" block :loading="loading">Create account</UButton>
          </UForm>
          <p class="mt-4 text-center text-sm text-muted">
            Already have an account? <NuxtLink to="/admin/login" class="text-primary hover:underline">Sign in</NuxtLink>
          </p>
        </template>
      </div>
    </div>
  </UApp>
</template>
