package booking

import (
	"testing"
	"time"
)

func TestWeekSlotsAcrossMidnightClockChange(t *testing.T) {
	location, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Fatal(err)
	}

	dates := [...]string{
		"2026-09-05", "2026-09-06", "2026-09-07", "2026-09-08",
		"2026-09-09", "2026-09-10", "2026-09-11", "2026-09-12",
	}
	const weekDays, slotsPerDay = 7, 9
	for start, today := range dates[:2] {
		t.Run(today, func(t *testing.T) {
			now, err := time.ParseInLocation(time.DateTime, today+" 12:00:00", location)
			if err != nil {
				t.Fatal(err)
			}
			slots := WeekSlots(now, location)
			if len(slots) != weekDays*slotsPerDay {
				t.Fatalf("got %d slots, want %d", len(slots), weekDays*slotsPerDay)
			}

			for day, wantDate := range dates[start : start+weekDays] {
				daySlots := slots[day*slotsPerDay : (day+1)*slotsPerDay]
				for _, slot := range daySlots {
					if got := slot.StartsAt.In(location).Format(time.DateOnly); got != wantDate {
						t.Fatalf("day %d: got date %s, want %s", day+1, got, wantDate)
					}
				}
				first := daySlots[0].StartsAt.In(location).Format(time.TimeOnly)
				last := daySlots[slotsPerDay-1].EndsAt.In(location).Format(time.TimeOnly)
				if first != "09:00:00" || last != "18:00:00" {
					t.Fatalf("date %s: got hours %s–%s, want 09:00–18:00", wantDate, first, last)
				}
			}
		})
	}
}
