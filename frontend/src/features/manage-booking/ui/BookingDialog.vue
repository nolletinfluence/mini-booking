<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import { clockTime, dateLabels } from '@/entities/slot'
import { ApiError, retryMessage } from '@/shared/api/http'
import { cancelBooking, createBooking } from '../api/bookings'
import type { BookingAction } from '../model/action'

const action = defineModel<BookingAction | null>({ required: true })
const emit = defineEmits<{ resolved: [message: string]; conflict: []; closed: [] }>()

const name = ref('')
const email = ref('')
const busy = ref(false)
const conflicted = ref(false)
const error = ref('')
const fieldErrors = ref<Record<string, string[]>>({})

const creating = computed(() => action.value?.kind === 'create')
const title = computed(() => creating.value ? 'Забронировать переговорную' : 'Отменить бронь?')
const date = computed(() => action.value && dateLabels(action.value.slot.starts_at.slice(0, 10)))
const submitLabel = computed(() => {
  if (busy.value) return creating.value ? 'Бронируем…' : 'Отменяем…'
  return creating.value ? 'Забронировать' : 'Отменить бронь'
})

watch(action, current => {
  if (!current) return
  name.value = ''
  email.value = ''
  error.value = ''
  fieldErrors.value = {}
  conflicted.value = false
})

function close() {
  if (!busy.value) action.value = null
}

function resolve(message: string) {
  action.value = null
  emit('resolved', message)
}

async function submit() {
  const current = action.value
  if (!current || busy.value || conflicted.value) return

  busy.value = true
  error.value = ''
  fieldErrors.value = {}

  try {
    if (current.kind === 'create') {
      await createBooking(current.slot.id, name.value.trim(), email.value.trim())
      resolve('Переговорная забронирована')
    } else {
      await cancelBooking(current.bookingId)
      resolve('Бронь отменена')
    }
  } catch (cause) {
    if (cause instanceof ApiError) {
      if (cause.status === 422) {
        fieldErrors.value = cause.fields ?? {}
        error.value = fieldErrors.value.slot_id?.[0] || 'Проверьте данные бронирования.'
        return
      }
      if (cause.status === 409) {
        error.value = cause.message
        conflicted.value = true
        emit('conflict')
        return
      }
      if (cause.status === 404 && current.kind === 'cancel') {
        resolve('Бронь уже отменена')
        return
      }
    }
    const failure = current.kind === 'create' ? 'Не удалось забронировать.' : 'Не удалось отменить бронь.'
    error.value = `${failure} ${retryMessage(cause)}`
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <Dialog
    :visible="action !== null"
    :header="title"
    :closable="!busy"
    :close-on-escape="!busy"
    :draggable="false"
    :close-button-props="{ class: 'icon-button' }"
    :pt="{
      root: 'booking-dialog',
      mask: 'booking-dialog-mask',
      header: 'dialog-header',
      title: 'dialog-title',
      content: 'dialog-content',
    }"
    modal
    @update:visible="close"
    @after-hide="emit('closed')"
  >
    <form v-if="action && date" class="booking-form" :aria-busy="busy" @submit.prevent="submit">
      <p class="dialog-time">
        <span class="pi pi-calendar" aria-hidden="true"></span>
        <time :datetime="action.slot.starts_at">
          {{ date.weekday }}, {{ date.full }} · {{ clockTime(action.slot.starts_at) }}–{{ clockTime(action.slot.ends_at) }}
        </time>
      </p>

      <template v-if="creating">
        <div class="form-field">
          <label for="customer-name">Имя</label>
          <InputText
            id="customer-name"
            v-model="name"
            class="field-input"
            autocomplete="name"
            required
            pattern=".*\S.*"
            :disabled="busy || conflicted"
            :invalid="Boolean(fieldErrors.customer_name)"
            :aria-invalid="Boolean(fieldErrors.customer_name)"
            :aria-describedby="fieldErrors.customer_name ? 'customer-name-error' : undefined"
            autofocus
            @update:model-value="delete fieldErrors.customer_name"
          />
          <small v-if="fieldErrors.customer_name" id="customer-name-error" class="field-error">
            {{ fieldErrors.customer_name[0] }}
          </small>
        </div>
        <div class="form-field">
          <label for="customer-email">Email</label>
          <InputText
            id="customer-email"
            v-model="email"
            class="field-input"
            type="email"
            autocomplete="email"
            required
            :disabled="busy || conflicted"
            :invalid="Boolean(fieldErrors.customer_email)"
            :aria-invalid="Boolean(fieldErrors.customer_email)"
            :aria-describedby="fieldErrors.customer_email ? 'customer-email-error' : undefined"
            @update:model-value="delete fieldErrors.customer_email"
          />
          <small v-if="fieldErrors.customer_email" id="customer-email-error" class="field-error">
            {{ fieldErrors.customer_email[0] }}
          </small>
        </div>
      </template>
      <p v-else class="dialog-description">После отмены будущий слот снова будет доступен для бронирования.</p>

      <p v-if="error" class="dialog-error" role="alert">{{ error }}</p>

      <div class="dialog-actions">
        <Button
          type="button"
          class="control-button neutral-button"
          :label="conflicted ? 'К расписанию' : creating ? 'Закрыть' : 'Оставить бронь'"
          :disabled="busy"
          :autofocus="!creating"
          @click="close"
        />
        <Button
          v-if="!conflicted"
          type="submit"
          class="control-button"
          :class="creating ? 'primary-button' : 'danger-button'"
          :label="submitLabel"
          :disabled="busy"
          :loading="busy"
        />
      </div>
    </form>
  </Dialog>
</template>
