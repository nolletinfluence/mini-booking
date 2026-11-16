import { request } from '@/shared/api/http'
import type { Schedule } from '../model/slot'

export function getSchedule() {
  return request<Schedule>('/slots')
}
