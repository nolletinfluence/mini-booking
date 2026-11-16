<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import Button from 'primevue/button'
import Skeleton from 'primevue/skeleton'
import { clockTime, statusLabels } from '@/entities/slot'
import type { Slot } from '@/entities/slot'
import { BookingDialog } from '@/features/manage-booking'
import type { BookingAction } from '@/features/manage-booking'
import { useSchedule } from '../model/useSchedule'

const { schedule, days, activeDay, selectedDate, loading, error, refresh } = useSchedule()
const action = ref<BookingAction | null>(null)
const notice = ref('')
const dayHeading = ref<HTMLElement>()
const dateList = ref<HTMLElement>()
let actionTrigger: HTMLButtonElement | null = null

const openSlots = computed(() => activeDay.value?.slots.filter(slot => slot.status === 'available').length ?? 0)
const countWord = new Intl.PluralRules('ru-RU')
const openSlotsLabel = computed(() => {
  const count = openSlots.value
  if (count === 0) return 'Нет свободных слотов'
  if (countWord.select(count) === 'one') return `${count} свободный слот`
  if (countWord.select(count) === 'few') return `${count} свободных слота`
  return `${count} свободных слотов`
})

onMounted(async () => {
  await refresh()
  await nextTick()
  const selected = dateList.value?.querySelector('[aria-pressed="true"]')
  selected?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
})

function selectDate(date: string) {
  selectedDate.value = date
  notice.value = ''
}

function manageSlot(slot: Slot, event: Event) {
  if (loading.value || error.value || action.value) return
  if (slot.booking_id != null) action.value = { kind: 'cancel', slot, bookingId: slot.booking_id }
  else if (slot.status === 'available') action.value = { kind: 'create', slot }

  notice.value = ''
  actionTrigger = event.currentTarget as HTMLButtonElement
}

function onResolved(message: string) {
  notice.value = message
  void refresh()
}

function restoreFocus() {
  if (actionTrigger?.isConnected && !actionTrigger.disabled) {
    actionTrigger.focus({ preventScroll: true })
  } else {
    dayHeading.value?.focus({ preventScroll: true })
  }
  actionTrigger = null
}
</script>

<template>
  <header class="masthead">
    <div class="site-width masthead-inner">
      <div class="wordmark">
        <span class="wordmark-mark"><span class="pi pi-calendar" aria-hidden="true"></span></span>
        <span>Mini Booking</span>
      </div>
      <span v-if="schedule" class="time-zone">
        <span class="pi pi-clock" aria-hidden="true"></span>
        <span>{{ schedule.time_zone }}</span>
      </span>
    </div>
  </header>

  <main class="site-width main-content">
    <div class="page-intro">
      <h1>Переговорная</h1>
      <p>Каждый день, с 09:00 до 18:00</p>
    </div>

    <div class="schedule-layout">
      <nav v-if="days.length" class="date-filter" aria-labelledby="date-filter-title">
        <h2 id="date-filter-title" class="date-filter-title">Выберите дату</h2>
        <ul ref="dateList" class="date-list">
          <li v-for="day in days" :key="day.key">
            <Button
              type="button"
              class="date-option"
              :aria-label="`${day.weekday}, ${day.full}${day.isToday ? ', сегодня' : ''}`"
              :aria-pressed="selectedDate === day.key"
              aria-controls="day-schedule"
              @click="selectDate(day.key)"
            >
              <span class="date-option-day">{{ day.day }}</span>
              <span class="date-option-full">
                <span class="date-option-weekday">{{ day.weekday }}</span>
                <span class="date-option-date">{{ day.month }}</span>
              </span>
              <span class="date-option-short">{{ day.shortWeekday }}</span>
              <span class="date-option-month">{{ day.shortMonth }}</span>
              <span v-if="day.isToday" class="date-option-today">Сегодня</span>
            </Button>
          </li>
        </ul>
      </nav>

      <section
        id="day-schedule"
        class="day-schedule"
        :class="{ 'day-schedule-full': !days.length }"
        aria-labelledby="day-schedule-title"
      >
        <div v-if="activeDay" class="agenda-heading">
          <div>
            <h2 id="day-schedule-title" ref="dayHeading" tabindex="-1">
              <time :datetime="activeDay.key">{{ activeDay.full }}</time>
            </h2>
            <p class="agenda-weekday">{{ activeDay.weekday }}</p>
          </div>

          <div class="availability">
            <span class="availability-label">{{ loading ? 'Обновляем расписание…' : openSlotsLabel }}</span>
            <div class="availability-strip" :class="{ 'availability-updating': loading }" aria-hidden="true">
              <span v-for="slot in activeDay.slots" :key="slot.id" :class="'bar-' + slot.status"></span>
            </div>
          </div>
        </div>
        <h2 v-else id="day-schedule-title" class="standalone-heading">Расписание</h2>

        <p class="sr-only" role="status">{{ notice || (loading ? 'Загружаем расписание…' : '') }}</p>
        <p v-if="notice" class="schedule-message schedule-success" aria-hidden="true">
          <span class="pi pi-check-circle" aria-hidden="true"></span>
          {{ notice }}
        </p>
        <div v-if="error" class="schedule-message schedule-error">
          <p role="alert">{{ error }}</p>
          <Button type="button" class="control-button neutral-button" label="Повторить" @click="refresh" />
        </div>

        <table v-if="activeDay" class="slots-table" :aria-busy="loading">
          <thead>
            <tr><th scope="col">Время</th><th scope="col">Статус</th><th scope="col"><span class="sr-only">Действие</span></th></tr>
          </thead>
          <tbody>
            <tr v-for="slot in activeDay.slots" :key="slot.id" :class="'slot-' + slot.status">
              <th scope="row" class="slot-time">
                <time :datetime="slot.starts_at">{{ clockTime(slot.starts_at) }}</time>
                <span class="slot-separator" aria-label="до">–</span>
                <time :datetime="slot.ends_at">{{ clockTime(slot.ends_at) }}</time>
              </th>
              <td>
                <span class="slot-status" :class="'status-' + slot.status">{{ statusLabels[slot.status] }}</span>
              </td>
              <td class="slot-action">
                <Button
                  v-if="slot.booking_id != null || slot.status === 'available'"
                  type="button"
                  class="control-button slot-button"
                  :class="slot.booking_id == null ? 'book-button' : 'cancel-button'"
                  :label="slot.booking_id == null ? 'Забронировать' : 'Отменить'"
                  :aria-label="`${slot.booking_id == null ? 'Забронировать' : 'Отменить бронь'} на ${clockTime(slot.starts_at)}`"
                  :disabled="loading || Boolean(error)"
                  @click="manageSlot(slot, $event)"
                />
                <span v-else class="no-action" aria-hidden="true">—</span>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-if="!activeDay && loading" class="skeleton-list" aria-hidden="true">
          <Skeleton v-for="line in 5" :key="line" class="skeleton-line" />
        </div>

        <div v-if="!activeDay && !loading && !error" class="empty-schedule">
          <span class="pi pi-calendar" aria-hidden="true"></span>
          <p>В расписании пока нет слотов.</p>
          <Button type="button" class="control-button neutral-button" label="Обновить" @click="refresh" />
        </div>
      </section>
    </div>
  </main>

  <BookingDialog v-model="action" @resolved="onResolved" @conflict="refresh" @closed="restoreFocus" />
</template>
