<script setup lang="ts">
import type { RestaurantDetail, Visit } from '~~/shared/types'

const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ restaurantId: number, visit: Visit | null }>()
const emit = defineEmits<{ saved: [detail: RestaurantDetail] }>()

const toast = useToast()
const state = reactive({ visited_on: todayISO(), meal: mealNow(), notes: '' })
const saving = ref(false)

watch(open, (isOpen) => {
  if (!isOpen) return
  Object.assign(state, props.visit
    ? { visited_on: props.visit.visited_on, meal: props.visit.meal, notes: props.visit.notes }
    : { visited_on: todayISO(), meal: mealNow(), notes: '' })
})

const meals = [{ label: 'Lunch', value: 'lunch' }, { label: 'Dinner', value: 'dinner' }]

async function submit() {
  saving.value = true
  try {
    const detail = await api<RestaurantDetail>(
      props.visit ? `/api/admin/visits/${props.visit.id}` : `/api/admin/restaurants/${props.restaurantId}/visits`,
      { method: props.visit ? 'PUT' : 'POST', body: { ...state } }
    )
    emit('saved', detail)
    open.value = false
  } catch (e) {
    toast.add({ title: 'Could not save visit', description: errorMessage(e), color: 'error' })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" :title="visit ? 'Edit visit' : 'Add a visit'" description="When did you go?">
    <template #body>
      <UForm :state="state" class="space-y-4" @submit="submit">
        <div class="grid grid-cols-2 gap-4">
          <UFormField label="Date" name="visited_on" required>
            <DateInput v-model="state.visited_on" />
          </UFormField>
          <UFormField label="Meal" name="meal">
            <USelect v-model="state.meal" :items="meals" class="w-full" />
          </UFormField>
        </div>
        <UFormField label="Notes" name="notes">
          <UTextarea v-model="state.notes" :rows="3" autoresize class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="open = false">Cancel</UButton>
          <UButton type="submit" :loading="saving">Save visit</UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>
