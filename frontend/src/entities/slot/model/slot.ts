export type SlotStatus = 'available' | 'booked' | 'past'

export interface Slot {
  id: number
  starts_at: string
  ends_at: string
  status: SlotStatus
  booking_id?: number
}

export interface Schedule {
  time_zone: string
  today: string
  slots: Slot[]
}

export const statusLabels: Record<SlotStatus, string> = {
  available: 'Свободно',
  booked: 'Занято',
  past: 'Прошло',
}
