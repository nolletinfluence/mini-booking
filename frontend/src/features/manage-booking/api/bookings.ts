import { request } from '@/shared/api/http'

export function createBooking(slotId: number, customerName: string, customerEmail: string) {
  return request<{ id: number }>('/bookings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      slot_id: slotId,
      customer_name: customerName,
      customer_email: customerEmail,
    }),
  })
}

export function cancelBooking(bookingId: number) {
  return request<void>(`/bookings/${bookingId}`, { method: 'DELETE' })
}
