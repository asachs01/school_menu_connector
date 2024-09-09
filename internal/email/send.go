package email

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/asachs01/school_menu_connector/internal/menu"
)

// Send sends an email with the given parameters
func Send(smtpServer, from, password string, to []string, subject, body string) error {
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

func SendLunchMenu(buildingID, districtID, startDate, endDate string, recipients, smtpServer, sender, password, subject string, debug bool) error {
	menuData, err := menu.Fetch(buildingID, districtID, startDate, endDate, debug)
	if err != nil {
		return fmt.Errorf("fetching menu: %w", err)
	}

	lunchMenu := menuData.GetLunchMenuString()
	if lunchMenu == "" {
		return fmt.Errorf("no lunch menu found for the specified date range")
	}

	recipientList := strings.Split(recipients, ",")
	for i, email := range recipientList {
		recipientList[i] = strings.TrimSpace(email)
	}

	if subject == "" {
		subject = fmt.Sprintf("Lunch Menu (%s - %s)", startDate, endDate)
	}

	if debug {
		fmt.Printf("Sending email to %s with subject: %s\n", recipients, subject)
	}

	if err := Send(smtpServer, sender, password, recipientList, subject, lunchMenu); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}
