<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import type { AdminAbout } from '~~/shared/types'

definePageMeta({ layout: 'admin' })
useSeoMeta({ title: `About page — ${SITE_NAME} admin`, robots: 'noindex' })

const toast = useToast()
const { data: about } = await useFetch<AdminAbout>('/api/admin/about')

const text = ref(about.value?.markdown ?? '')
const dirty = computed(() => text.value.trim() !== (about.value?.markdown ?? '').trim())
const saving = ref(false)

const placeholders = [
  { code: '{{names}}', help: 'everyone who has scored a restaurant, e.g. “Benoit & Sarvin”' },
  { code: '{{restaurants}}', help: 'how many restaurants, e.g. “12 restaurants”' }
]

async function save() {
  saving.value = true
  try {
    about.value = await api<AdminAbout>('/api/admin/about', { method: 'PUT', body: { markdown: text.value } })
    text.value = about.value.markdown
    toast.add({ title: 'About page saved', color: 'success' })
  } catch (e) {
    toast.add({ title: 'Could not save', description: errorMessage(e), color: 'error' })
  } finally {
    saving.value = false
  }
}

// The preview is rendered by the server, exactly as the public page will be.
const preview = ref('')
async function renderPreview() {
  try {
    preview.value = (await api<{ html: string }>('/api/admin/about/preview', { method: 'POST', body: { markdown: text.value } })).html
  } catch {
    // Keep the last preview; saving reports any real problem.
  }
}
watchDebounced(text, renderPreview, { debounce: 400, immediate: true })

const resetOpen = ref(false)
function resetToDefault() {
  text.value = about.value?.default_markdown ?? ''
  resetOpen.value = false
}

// Photo
const file = ref<File | null>(null)
const uploading = ref(false)
watch(file, async (f) => {
  if (!f) return
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('file', f)
    const res = await api<AdminAbout>('/api/admin/about/photo', { method: 'POST', body: fd })
    if (about.value) about.value.photo = res.photo
  } catch (e) {
    toast.add({ title: 'Upload failed', description: errorMessage(e), color: 'error' })
  } finally {
    file.value = null
    uploading.value = false
  }
})

const removeOpen = ref(false)
async function removePhoto() {
  removeOpen.value = false
  try {
    const res = await api<AdminAbout>('/api/admin/about/photo', { method: 'DELETE' })
    if (about.value) about.value.photo = res.photo
  } catch (e) {
    toast.add({ title: 'Could not remove photo', description: errorMessage(e), color: 'error' })
  }
}

onBeforeRouteLeave(() => !dirty.value || window.confirm('Leave without saving the About text?'))
</script>

<template>
  <UDashboardPanel>
    <template #header>
      <UDashboardNavbar title="About page">
        <template #right>
          <UButton to="/about" color="neutral" variant="ghost" size="sm" icon="i-lucide-eye">View</UButton>
          <UButton size="sm" icon="i-lucide-check" :loading="saving" :disabled="!dirty" @click="save">Save</UButton>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="space-y-12">
        <section class="grid gap-6 xl:grid-cols-2">
          <div class="space-y-3">
            <div class="flex items-center justify-between gap-2">
              <h2 class="font-display text-xl">Text</h2>
              <div class="flex items-center gap-2">
                <UBadge v-if="about?.is_default && !dirty" color="neutral" variant="subtle">Default text</UBadge>
                <UButton
                  v-if="text.trim() !== about?.default_markdown.trim()"
                  color="neutral"
                  variant="ghost"
                  size="sm"
                  icon="i-lucide-rotate-ccw"
                  @click="resetOpen = true"
                >
                  Reset to default
                </UButton>
              </div>
            </div>
            <UTextarea v-model="text" autoresize :rows="16" :maxrows="40" class="w-full" :ui="{ base: 'font-mono text-sm' }" />
            <div class="space-y-1 text-sm text-muted">
              <p>
                Markdown: <code>**bold**</code>, <code>*italic*</code>, <code>## Heading</code>,
                <code>[link](https://…)</code>, <code>- list</code>. Leave a blank line between paragraphs.
              </p>
              <p v-for="p in placeholders" :key="p.code"><code class="text-highlighted">{{ p.code }}</code> is replaced with {{ p.help }}.</p>
            </div>
          </div>

          <div class="space-y-3">
            <h2 class="font-display text-xl">Preview</h2>
            <div class="journal rounded-lg border border-default p-6">
              <div class="prose-journal text-lg leading-relaxed text-toned" v-html="preview" />
            </div>
          </div>
        </section>

        <section class="max-w-xl space-y-3">
          <h2 class="font-display text-xl">Photo</h2>
          <p class="text-sm text-muted">Optional. Shown above the text on the About page.</p>
          <div v-if="about?.photo" class="space-y-2">
            <img :src="mediaUrl(about.photo, 'full')" alt="About page photo" class="max-h-72 rounded-md object-cover">
            <UButton color="error" variant="ghost" size="sm" icon="i-lucide-trash-2" @click="removeOpen = true">Remove photo</UButton>
          </div>
          <UFileUpload
            v-model="file"
            accept="image/jpeg,image/png,image/webp"
            :label="about?.photo ? 'Replace the photo' : 'Drop a photo here or click to browse'"
            description="JPEG, PNG or WebP, up to 20 MB."
            :disabled="uploading"
            class="min-h-24"
          />
          <p v-if="uploading" class="text-sm text-muted">Uploading…</p>
        </section>
      </div>

      <ConfirmModal
        v-model:open="resetOpen"
        title="Reset to the default text?"
        description="Your text is replaced in the editor. Nothing changes on the site until you save."
        confirm-label="Reset"
        @confirm="resetToDefault"
      />
      <ConfirmModal v-model:open="removeOpen" title="Remove photo?" description="This can't be undone." @confirm="removePhoto" />
    </template>
  </UDashboardPanel>
</template>
