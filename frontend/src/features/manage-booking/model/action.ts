import type { Slot } from '@/entities/slot'

export type BookingAction =
  | { kind: 'create'; slot: Slot }
  | { kind: 'cancel'; slot: Slot; bookingId: number }
