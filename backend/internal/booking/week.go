package booking

import "time"

func WeekSlots(now time.Time, location *time.Location) []Slot {
	const firstHour, closingHour, days = 9, 18, 7
	year, month, day := now.In(location).Date()
	firstCalendarDate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	slots := make([]Slot, 0, days*(closingHour-firstHour))
	for dayOffset := range days {
		year, month, day := firstCalendarDate.AddDate(0, 0, dayOffset).Date()
		for hour := firstHour; hour < closingHour; hour++ {
			start := time.Date(year, month, day, hour, 0, 0, 0, location)
			slots = append(slots, Slot{StartsAt: start, EndsAt: start.Add(time.Hour)})
		}
	}
	return slots
}
