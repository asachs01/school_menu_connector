/*
LINQ Connect Menu Notifier
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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

const apiURL = "https://api.linqconnect.com/api/FamilyMenu"

// MenuResponse represents the structure of the API response
type MenuResponse struct {
	FamilyMenuSessions []struct {
		ServingSession string `json:"ServingSession"`
		MenuPlans      []struct {
			Days []struct {
				Date      string `json:"Date"`
				MenuMeals []struct {
					MenuMealName     string `json:"MenuMealName"`
					RecipeCategories []struct {
						CategoryName string `json:"CategoryName"`
						Recipes      []struct {
							RecipeName string `json:"RecipeName"`
						} `json:"Recipes"`
					} `json:"RecipeCategories"`
				} `json:"MenuMeals"`
			} `json:"Days"`
		} `json:"MenuPlans"`
	} `json:"FamilyMenuSessions"`
}

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
	debugFlag := flag.Bool("debug", false, "Enable debug output")
	flag.Parse()

	if err := run(*buildingID, *districtID, *recipients, *sender, *password, *smtpServer, *subject, *startDate, *endDate, *weekStart, *icsOutputPath, *emailFlag, *icsFlag, *debugFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(buildingID, districtID, recipients, sender, password, smtpServer, subject, startDate, endDate, weekStart, icsOutputPath string, emailFlag, icsFlag, debugFlag bool) error {
	if !emailFlag && !icsFlag {
		return fmt.Errorf("at least one of -email or -ics must be provided")
	}

	if buildingID == "" || districtID == "" {
		return fmt.Errorf("building ID and district ID are required for all operations")
	}

	if emailFlag {
		if recipients == "" || sender == "" || password == "" {
			return fmt.Errorf("recipient email, sender email, and password are required for email")
		}
	}

	// Set default dates if not provided
	today := time.Now().Format("01-02-2006")
	if startDate == "" {
		startDate = today
	}
	if endDate == "" {
		endDate = startDate
	}

	url := constructURL(buildingID, districtID, startDate, endDate)
	menu, err := getMenu(url)
	if err != nil {
		return fmt.Errorf("fetching menu: %w", err)
	}

	lunchMenu := getLunchMenuString(menu)
	if lunchMenu == "" {
		return fmt.Errorf("no lunch menu found for the specified date range")
	}

	if emailFlag {
		if smtpServer == "" {
			smtpServer = "smtp.gmail.com:587"
		}
		if subject == "" {
			subject = "Lunch Menu"
		}
		// Update subject with date or date range
		if startDate == endDate {
			subject = fmt.Sprintf("%s (%s)", subject, startDate)
		} else {
			subject = fmt.Sprintf("%s (%s to %s)", subject, startDate, endDate)
		}

		recipientList := strings.Split(recipients, ",")
		for i, email := range recipientList {
			recipientList[i] = strings.TrimSpace(email)
		}

		if err := sendEmail(smtpServer, sender, password, recipientList, subject, lunchMenu); err != nil {
			return fmt.Errorf("sending email: %w", err)
		}

		fmt.Println("Lunch menu sent successfully!")
	}

	if icsFlag {
		if icsOutputPath == "" {
			// Parse the weekStart date
			startDate, err := time.Parse("01-02-2006", weekStart)
			if err != nil {
				return fmt.Errorf("invalid week start date: %w", err)
			}
			// Create the filename with the date
			icsOutputPath = fmt.Sprintf("lunch_menu_%s.ics", startDate.Format("01-02-2006"))
		}
		if weekStart == "" {
			weekStart = startDate
		}
		if err := createICSFile(buildingID, districtID, weekStart, icsOutputPath, debugFlag); err != nil {
			return fmt.Errorf("creating ICS file: %w", err)
		}
		fmt.Printf("ICS file created at: %s\n", icsOutputPath)
	}

	return nil
}

func constructURL(buildingID, districtID, startDate, endDate string) string {
	return fmt.Sprintf("%s?buildingId=%s&districtId=%s&startDate=%s&endDate=%s",
		apiURL, buildingID, districtID, startDate, endDate)
}

func getMenu(url string) (*MenuResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var menu MenuResponse
	if err := json.Unmarshal(body, &menu); err != nil {
		return nil, err
	}

	return &menu, nil
}

func getLunchMenuString(menu *MenuResponse) string {
	var lunchMenu strings.Builder

	for _, session := range menu.FamilyMenuSessions {
		if session.ServingSession == "Lunch" {
			for _, plan := range session.MenuPlans {
				for _, day := range plan.Days {
					fmt.Fprintf(&lunchMenu, "Lunch Menu for %s:\n\n", day.Date)
					for _, meal := range day.MenuMeals {
						fmt.Fprintf(&lunchMenu, "%s:\n", meal.MenuMealName)
						for _, category := range meal.RecipeCategories {
							fmt.Fprintf(&lunchMenu, "  %s:\n", category.CategoryName)
							for _, recipe := range category.Recipes {
								fmt.Fprintf(&lunchMenu, "    - %s\n", recipe.RecipeName)
							}
						}
						lunchMenu.WriteString("\n")
					}
					lunchMenu.WriteString("\n")
				}
			}
			return lunchMenu.String()
		}
	}
	return ""
}

func sendEmail(smtpServer, from, password string, to []string, subject, body string) error {
	// Use the first recipient as the main "To" address
	mainRecipient := to[0]
	bcc := strings.Join(to[1:], ", ")

	header := make(map[string]string)
	header["From"] = from
	header["To"] = mainRecipient
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	if bcc != "" {
		header["Bcc"] = bcc
	}

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	auth := smtp.PlainAuth("", from, password, strings.Split(smtpServer, ":")[0])

	return smtp.SendMail(smtpServer, auth, from, to, []byte(message))
}

func createICSFile(buildingID, districtID, weekStart, outputPath string, debug bool) error {
	// Parse the week start date
	start, err := time.Parse("01-02-2006", weekStart)
	if err != nil {
		return fmt.Errorf("invalid week start date: %w", err)
	}

	// Create a new calendar
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)

	// Iterate through the menu for 5 days (Monday to Friday)
	for i := 0; i < 5; i++ {
		date := start.AddDate(0, 0, i)
		dateStr := date.Format("01-02-2006")

		// Fetch menu for this specific date
		url := constructURL(buildingID, districtID, dateStr, dateStr)
		menu, err := getMenu(url)
		if err != nil {
			if debug {
				fmt.Printf("Error fetching menu for date %s: %v\n", dateStr, err)
			}
			continue
		}

		lunchMenu := getLunchMenuForDate(menu, date.Format("1/2/2006"), debug)

		if lunchMenu != "" {
			event := cal.AddEvent(fmt.Sprintf("lunch-%s", date.Format("2006-01-02")))
			event.SetCreatedTime(time.Now())
			event.SetDtStampTime(time.Now())
			event.SetModifiedAt(time.Now())

			// Set as an all-day event
			event.SetAllDayStartAt(date)
			event.SetAllDayEndAt(date.AddDate(0, 0, 1)) // End date is exclusive, so we add one day

			event.SetSummary(fmt.Sprintf("Lunch Menu - %s", date.Format("01/02/2006")))
			event.SetDescription(lunchMenu)
		} else if debug {
			fmt.Printf("No lunch menu found for date: %s\n", date.Format("01/02/2006"))
		}
	}

	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write the ICS file
	return cal.SerializeTo(file)
}

func getLunchMenuForDate(menu *MenuResponse, date string, debug bool) string {
	var lunchMenu strings.Builder

	if debug {
		fmt.Printf("Searching for lunch menu on date: %s\n", date)
	}

	for _, session := range menu.FamilyMenuSessions {
		if session.ServingSession == "Lunch" {
			for _, plan := range session.MenuPlans {
				for _, day := range plan.Days {
					if debug {
						fmt.Printf("Checking day: %s\n", day.Date)
					}

					if day.Date == date {
						// Build lunch menu
						fmt.Fprintf(&lunchMenu, "Lunch Menu for %s:\n\n", day.Date)
						for _, meal := range day.MenuMeals {
							fmt.Fprintf(&lunchMenu, "%s:\n", meal.MenuMealName)
							for _, category := range meal.RecipeCategories {
								fmt.Fprintf(&lunchMenu, "  %s:\n", category.CategoryName)
								for _, recipe := range category.Recipes {
									fmt.Fprintf(&lunchMenu, "    - %s\n", recipe.RecipeName)
								}
							}
							lunchMenu.WriteString("\n")
						}
						return lunchMenu.String()
					}
				}
			}
		}
	}

	if debug {
		fmt.Printf("No lunch menu found for date: %s\n", date)
	}
	return ""
}
