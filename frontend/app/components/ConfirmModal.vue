<script setup lang="ts">
const open = defineModel<boolean>('open', { default: false })
withDefaults(defineProps<{ title: string, description?: string, confirmLabel?: string, loading?: boolean }>(), { confirmLabel: 'Delete' })
defineEmits<{ confirm: [] }>()
</script>

<template>
  <UModal v-model:open="open" :title="title" :description="description">
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="ghost" @click="open = false">Cancel</UButton>
        <UButton color="error" :loading="loading" @click="$emit('confirm')">{{ confirmLabel }}</UButton>
      </div>
    </template>
  </UModal>
</template>
