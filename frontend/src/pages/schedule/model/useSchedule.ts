import { computed, ref } from 'vue'
import { dateLabels, getSchedule } from '@/entities/slot'
import type { Schedule, Slot } from '@/entities/slot'
import { retryMessage } from '@/shared/api/http'

export function useSchedule() {
  const schedule = ref<Schedule>()
  const selectedDate = ref('')
  const loading = ref(false)
  const error = ref('')

  const days = computed(() => {
    const dates = new Map<string, Slot[]>()
    for (const slot of schedule.value?.slots ?? []) {
      const key = slot.starts_at.slice(0, 10)
      const slots = dates.get(key)
      if (slots) slots.push(slot)
      else dates.set(key, [slot])
    }

    return Array.from(dates, ([key, slots]) => ({
      key,
      ...dateLabels(key),
      isToday: key === schedule.value?.today,
      slots,
    }))
  })
  const activeDay = computed(() => days.value.find(day => day.key === selectedDate.value))

  async function refresh() {
    if (loading.value) return
    loading.value = true
    error.value = ''
    try {
      const loaded = await getSchedule()
      schedule.value = loaded
      if (!activeDay.value) {
        const nextDay = days.value.find(day => day.isToday)
          ?? days.value.find(day => day.key >= loaded.today)
          ?? days.value.at(-1)
        selectedDate.value = nextDay?.key ?? ''
      }
    } catch (cause) {
      error.value = `Не удалось загрузить расписание. ${retryMessage(cause)}`
    } finally {
      loading.value = false
    }
  }

  return { schedule, days, activeDay, selectedDate, loading, error, refresh }
}
