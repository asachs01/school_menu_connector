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
)

const apiURL = "https://api.linqconnect.com/api/FamilyMenu"

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
	startDate := flag.String("start", os.Getenv("START_DATE"), "Start date (MM-DD-YYYY)")
	endDate := flag.String("end", os.Getenv("END_DATE"), "End date (MM-DD-YYYY)")
	flag.Parse()

	// Set default SMTP server if not provided
	if *smtpServer == "" {
		*smtpServer = "smtp.gmail.com:587"
	}

	// Set default subject if not provided
	if *subject == "" {
		*subject = "Lunch Menu"
	}

	// Set default dates if not provided
	today := time.Now().Format("01-02-2006")
	if *startDate == "" {
		*startDate = today
	}
	if *endDate == "" {
		*endDate = *startDate
	}

	if *buildingID == "" || *districtID == "" || *recipients == "" || *sender == "" || *password == "" {
		fmt.Println("Error: Building ID, district ID, recipient email, sender email, and password are required")
		flag.PrintDefaults()
		fmt.Println("\nEnvironment variables:")
		fmt.Println("  BUILDING_ID: Building ID")
		fmt.Println("  DISTRICT_ID: District ID")
		fmt.Println("  RECIPIENT_EMAIL: Recipient email address")
		fmt.Println("  SENDER_EMAIL: Sender email address")
		fmt.Println("  EMAIL_PASSWORD: Sender email password")
		fmt.Println("  SMTP_SERVER: SMTP server and port (default: smtp.gmail.com:587)")
		fmt.Println("  EMAIL_SUBJECT: Email subject line (default: Lunch Menu)")
		fmt.Println("  START_DATE: Start date (MM-DD-YYYY)")
		fmt.Println("  END_DATE: End date (MM-DD-YYYY)")
		return
	}

	url := constructURL(*buildingID, *districtID, *startDate, *endDate)
	menu, err := getMenu(url)
	if err != nil {
		fmt.Printf("Error fetching menu: %v\n", err)
		return
	}

	lunchMenu := getLunchMenuString(menu)
	if lunchMenu == "" {
		fmt.Println("No lunch menu found for the specified date range.")
		return
	}

	// Update subject with date or date range
	if *startDate == *endDate {
		*subject = fmt.Sprintf("%s (%s)", *subject, *startDate)
	} else {
		*subject = fmt.Sprintf("%s (%s to %s)", *subject, *startDate, *endDate)
	}

	recipientList := strings.Split(*recipients, ",")
	for i, email := range recipientList {
		recipientList[i] = strings.TrimSpace(email)
	}

	err = sendEmail(*smtpServer, *sender, *password, recipientList, *subject, lunchMenu)
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
		return
	}

	fmt.Println("Lunch menu sent successfully!")
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
	err = json.Unmarshal(body, &menu)
	if err != nil {
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
					lunchMenu.WriteString(fmt.Sprintf("Lunch Menu for %s:\n\n", day.Date))
					for _, meal := range day.MenuMeals {
						lunchMenu.WriteString(fmt.Sprintf("%s:\n", meal.MenuMealName))
						for _, category := range meal.RecipeCategories {
							lunchMenu.WriteString(fmt.Sprintf("  %s:\n", category.CategoryName))
							for _, recipe := range category.Recipes {
								lunchMenu.WriteString(fmt.Sprintf("    - %s\n", recipe.RecipeName))
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
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", strings.Join(to, ", "), subject, body))

	auth := smtp.PlainAuth("", from, password, strings.Split(smtpServer, ":")[0])

	err := smtp.SendMail(smtpServer, auth, from, to, msg)
	if err != nil {
		return err
	}

	return nil
}
