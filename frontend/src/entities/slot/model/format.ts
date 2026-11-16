const fullDate = new Intl.DateTimeFormat('ru-RU', {
  day: 'numeric',
  month: 'long',
  timeZone: 'UTC',
})
const month = new Intl.DateTimeFormat('ru-RU', { month: 'long', timeZone: 'UTC' })
const shortMonth = new Intl.DateTimeFormat('ru-RU', { month: 'short', timeZone: 'UTC' })
const shortWeekday = new Intl.DateTimeFormat('ru-RU', { weekday: 'short', timeZone: 'UTC' })
const weekday = new Intl.DateTimeFormat('ru-RU', { weekday: 'long', timeZone: 'UTC' })

export function dateLabels(key: string) {
  const date = new Date(`${key}T00:00:00Z`)
  return {
    full: fullDate.format(date),
    weekday: weekday.format(date),
    shortWeekday: shortWeekday.format(date),
    month: month.format(date),
    shortMonth: shortMonth.format(date),
    day: date.getUTCDate(),
  }
}

export function clockTime(timestamp: string): string {
  return timestamp.slice(11, 16)
}
