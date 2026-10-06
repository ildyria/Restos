<script setup lang="ts">
definePageMeta({ layout: false })
useSeoMeta({ title: 'Sign in — Restos', robots: 'noindex' })

const route = useRoute()
const toast = useToast()
const { login, user, checked, refresh } = useAuth()

const state = reactive({ email: '', password: '' })
const loading = ref(false)

onMounted(async () => {
  if (!checked.value) await refresh()
  if (user.value) await navigateTo(safeNext())
})

function safeNext() {
  const n = route.query.next
  return typeof n === 'string' && n.startsWith('/admin') ? n : '/admin'
}

async function submit() {
  loading.value = true
  try {
    await login(state.email, state.password)
    await navigateTo(safeNext())
  } catch (e) {
    toast.add({ title: 'Could not sign in', description: errorMessage(e), color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UApp>
    <div class="grid min-h-screen place-items-center px-4">
      <div class="w-full max-w-sm">
        <NuxtLink to="/" class="font-display mb-8 block text-center text-3xl font-medium">Restos</NuxtLink>
        <UForm :state="state" class="space-y-4 rounded-lg border border-default p-6" @submit="submit">
          <UFormField label="Email" name="email">
            <UInput v-model="state.email" type="email" autocomplete="username" required autofocus class="w-full" />
          </UFormField>
          <UFormField label="Password" name="password">
            <UInput v-model="state.password" type="password" autocomplete="current-password" required class="w-full" />
          </UFormField>
          <UButton type="submit" block :loading="loading">Sign in</UButton>
        </UForm>
      </div>
    </div>
  </UApp>
</template>
