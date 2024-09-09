package ics

import (
	"fmt"
	"os"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/asachs01/school_menu_connector/internal/menu"
)

func GenerateICSFile(buildingID, districtID, startDateStr, endDateStr, outputPath string, debug bool) error {
	start, err := time.Parse("01-02-2006", startDateStr)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("01-02-2006", endDateStr)
	if err != nil {
		return fmt.Errorf("invalid end date: %w", err)
	}

	if debug {
		fmt.Printf("Creating ICS file for date range: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	}

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)

	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		dateStr := date.Format("01-02-2006")

		if debug {
			fmt.Printf("Fetching menu for date: %s\n", dateStr)
		}

		menu, err := menu.Fetch(buildingID, districtID, dateStr, dateStr, debug)
		if err != nil {
			if debug {
				fmt.Printf("Error fetching menu for date %s: %v\n", dateStr, err)
			}
			continue
		}

		lunchMenu := menu.GetLunchMenuForDate(date.Format("1/2/2006"), debug)

		if lunchMenu != "" {
			event := cal.AddEvent(fmt.Sprintf("lunch-%s", date.Format("2006-01-02")))
			event.SetCreatedTime(time.Now())
			event.SetDtStampTime(time.Now())
			event.SetModifiedAt(time.Now())
			event.SetAllDayStartAt(date)
			event.SetAllDayEndAt(date.AddDate(0, 0, 1))
			event.SetSummary(fmt.Sprintf("Lunch Menu - %s", date.Format("01/02/2006")))
			event.SetDescription(lunchMenu)

			if debug {
				fmt.Printf("Added event for date: %s\n", date.Format("2006-01-02"))
			}
		} else if debug {
			fmt.Printf("No lunch menu found for date: %s\n", date.Format("01/02/2006"))
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return cal.SerializeTo(file)
}
