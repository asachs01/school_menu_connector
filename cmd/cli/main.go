/*
School Menu Connector
Copyright (C) 2024 Aaron Sachs

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/asachs01/school_menu_connector/internal/menu"
	"github.com/asachs01/school_menu_connector/internal/ics"
	"github.com/asachs01/school_menu_connector/internal/email"
)

func main() {
	buildingID := flag.String("building", os.Getenv("BUILDING_ID"), "Building ID")
	districtID := flag.String("district", os.Getenv("DISTRICT_ID"), "District ID")
	recipients := flag.String("recipient", os.Getenv("RECIPIENT_EMAIL"), "Recipient email address(es), comma-separated")
	sender := flag.String("sender", os.Getenv("SENDER_EMAIL"), "Sender email address")
	password := flag.String("password", os.Getenv("EMAIL_PASSWORD"), "Sender email password")
	smtpServer := flag.String("smtp", os.Getenv("SMTP_SERVER"), "SMTP server and port")
	subject := flag.String("subject", os.Getenv("EMAIL_SUBJECT"), "Email subject line")
	startDate := flag.String("startDate", os.Getenv("START_DATE"), "Start date (MM-DD-YYYY)")
	endDate := flag.String("endDate", os.Getenv("END_DATE"), "End date (MM-DD-YYYY)")
	weekStart := flag.String("week-start", "", "Start date of the week (MM-DD-YYYY) for calendar file")
	icsOutputPath := flag.String("ics-output-path", "", "Output path for the ICS file")
	emailFlag := flag.Bool("email", false, "Send email")
	icsFlag := flag.Bool("ics", false, "Generate ICS file")
	debugFlag := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	if err := run(*buildingID, *districtID, *recipients, *sender, *password, *smtpServer, *subject, *startDate, *endDate, *weekStart, *icsOutputPath, *emailFlag, *icsFlag, *debugFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(buildingID, districtID, recipients, sender, password, smtpServer, subject, startDate, endDate, weekStart, icsOutputPath string, emailFlag, icsFlag, debugFlag bool) error {
	start, err := time.Parse("01-02-2006", startDate)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}

	end, err := time.Parse("01-02-2006", endDate)
	if err != nil {
		return fmt.Errorf("invalid end date: %w", err)
	}

	if debugFlag {
		fmt.Printf("Fetching menu for date range: %s to %s\n", start.Format("01-02-2006"), end.Format("01-02-2006"))
	}

	menuData, err := menu.Fetch(buildingID, districtID, start.Format("01-02-2006"), end.Format("01-02-2006"), debugFlag)
	if err != nil {
		return fmt.Errorf("fetching menu: %w", err)
	}

	lunchMenu := menuData.GetLunchMenuString()
	if lunchMenu == "" {
		return fmt.Errorf("no lunch menu found for the specified date range")
	}

	if debugFlag {
		fmt.Println("Lunch menu found:")
		fmt.Println(lunchMenu)
	}

	if emailFlag {
		if err := email.SendLunchMenu(buildingID, districtID, start.Format("01-02-2006"), end.Format("01-02-2006"), recipients, smtpServer, sender, password, subject, debugFlag); err != nil {
			return fmt.Errorf("sending email: %w", err)
		}
		fmt.Println("Lunch menu sent successfully!")
	}

	if icsFlag {
		outputPath := fmt.Sprintf("lunch_menu_%s_to_%s.ics", start.Format("01-02-2006"), end.Format("01-02-2006"))
		_, err := ics.GenerateICSFile(buildingID, districtID, startDate, endDate, outputPath, debugFlag)
		if err != nil {
			return fmt.Errorf("failed to generate ICS file: %v", err)
		}
		fmt.Printf("ICS file generated successfully: %s\n", outputPath)
	}

	return nil
}
