<script setup lang="ts">
import { CalendarDate, getLocalTimeZone, parseDate, today } from '@internationalized/date'

// A date picker whose weeks start on Monday (the native <input type="date"> follows the browser locale).
// The model is a plain YYYY-MM-DD string, like the API.
const model = defineModel<string>({ required: true })
defineProps<{ id?: string }>()

const open = ref(false)
const value = computed({
  get: () => {
    try {
      return model.value ? parseDate(model.value) : today(getLocalTimeZone())
    } catch {
      return today(getLocalTimeZone())
    }
  },
  set: (d: CalendarDate | undefined) => {
    if (!d) return
    model.value = d.toString()
    open.value = false
  }
})
const label = computed(() => model.value ? formatDate(model.value, { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' }) : 'Pick a date')
</script>

<template>
  <UPopover v-model:open="open">
    <UButton :id="id" color="neutral" variant="outline" icon="i-lucide-calendar" class="w-full justify-start font-normal">
      {{ label }}
    </UButton>
    <template #content>
      <UCalendar v-model="value" :week-starts-on="1" locale="en-GB" class="p-2" />
    </template>
  </UPopover>
</template>
