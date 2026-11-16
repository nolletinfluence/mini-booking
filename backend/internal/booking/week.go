package booking

import "time"

func WeekSlots(now time.Time, location *time.Location) []Slot {
	const firstHour, closingHour, days = 9, 18, 7
	localNow := now.In(location)
	firstDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)

	slots := make([]Slot, 0, days*(closingHour-firstHour))
	for dayOffset := range days {
		date := firstDay.AddDate(0, 0, dayOffset)
		for hour := firstHour; hour < closingHour; hour++ {
			start := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, location)
			slots = append(slots, Slot{StartsAt: start, EndsAt: start.Add(time.Hour)})
		}
	}
	return slots
}
