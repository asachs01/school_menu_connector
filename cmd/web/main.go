package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "runtime/debug"
    "strings"

    "github.com/asachs01/school_menu_connector/internal/menu"
    "github.com/asachs01/school_menu_connector/internal/ics"
    mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
)

// Create a custom logger
var (
    infoLog  = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
    errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
)

var senderEmail string

func init() {
    senderEmail = os.Getenv("SENDER_EMAIL")
    if senderEmail == "" {
        senderEmail = "noreply@schoolmenuconnector.com"
    }
}

func main() {
    // Serve static files
    fs := http.FileServer(http.Dir("./web"))
    http.Handle("/", fs)

    // API endpoint
    http.HandleFunc("/api/generate", logRequest(handleGenerate))

    infoLog.Println("Starting server on :8080")
    err := http.ListenAndServe(":8080", nil)
    errorLog.Fatal(err)
}

func logRequest(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        infoLog.Printf("%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL.RequestURI())
        next.ServeHTTP(w, r)
    }
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if r := recover(); r != nil {
            errorLog.Printf("panic: %v\n%s", r, debug.Stack())
            http.Error(w, "Internal server error", http.StatusInternalServerError)
        }
    }()

    if r.Method != http.MethodPost {
        errorLog.Printf("Method not allowed: %s", r.Method)
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var data map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        errorLog.Printf("Invalid request body: %v", err)
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    buildingID, ok := data["buildingId"].(string)
    if !ok {
        errorLog.Print("Missing or invalid buildingId")
        http.Error(w, "Missing or invalid buildingId", http.StatusBadRequest)
        return
    }

    districtID, ok := data["districtId"].(string)
    if !ok {
        errorLog.Print("Missing or invalid districtId")
        http.Error(w, "Missing or invalid districtId", http.StatusBadRequest)
        return
    }

    startDate, ok := data["startDate"].(string)
    if !ok {
        errorLog.Print("Missing or invalid startDate")
        http.Error(w, "Missing or invalid startDate", http.StatusBadRequest)
        return
    }

    endDate, ok := data["endDate"].(string)
    if !ok {
        errorLog.Print("Missing or invalid endDate")
        http.Error(w, "Missing or invalid endDate", http.StatusBadRequest)
        return
    }

    action, ok := data["action"].(string)
    if !ok {
        errorLog.Print("Missing or invalid action")
        http.Error(w, "Missing or invalid action", http.StatusBadRequest)
        return
    }

    menuData, err := menu.Fetch(buildingID, districtID, startDate, endDate, false)
    if err != nil {
        errorLog.Printf("Error fetching menu: %v", err)
        http.Error(w, fmt.Sprintf("Error fetching menu: %v", err), http.StatusInternalServerError)
        return
    }

    switch action {
    case "email":
        message, err := handleEmail(data, menuData)
        if err != nil {
            errorLog.Printf("Error handling email: %v", err)
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        response := map[string]string{"message": message}
        w.Header().Set("Content-Type", "application/json")
        if err := json.NewEncoder(w).Encode(response); err != nil {
            errorLog.Printf("Error encoding response: %v", err)
            http.Error(w, "Error encoding response", http.StatusInternalServerError)
        }
    case "ics":
        if err := handleICS(w, data, menuData); err != nil {
            errorLog.Printf("Error handling ICS: %v", err)
            http.Error(w, fmt.Sprintf("Error generating calendar file: %v", err), http.StatusInternalServerError)
        }
        // Don't write anything else here
    default:
        errorLog.Printf("Invalid action: %s", action)
        http.Error(w, "Invalid action", http.StatusBadRequest)
    }
}

func handleEmail(data map[string]interface{}, menuData *menu.Menu) (string, error) {
    recipients, ok := data["recipients"].(string)
    if !ok || recipients == "" {
        return "", fmt.Errorf("missing or invalid recipients")
    }
    
    recipientList := strings.Split(recipients, ",")
    
    mailjetClient := mailjet.NewMailjetClient(os.Getenv("MJ_APIKEY_PUBLIC"), os.Getenv("MJ_APIKEY_PRIVATE"))

    var recipientsV31 mailjet.RecipientsV31
    for _, recipient := range recipientList {
        recipientsV31 = append(recipientsV31, mailjet.RecipientV31{
            Email: strings.TrimSpace(recipient),
        })
    }

    messagesInfo := []mailjet.InfoMessagesV31{
        {
            From: &mailjet.RecipientV31{
                Email: senderEmail,
                Name: "School Menu Connector",
            },
            To: &recipientsV31,
            Subject: "School Menu",
            TextPart: menuData.GetLunchMenuString(),
            HTMLPart: "<h3>School Menu</h3><pre>" + menuData.GetLunchMenuString() + "</pre>",
        },
    }

    messages := mailjet.MessagesV31{Info: messagesInfo}
    res, err := mailjetClient.SendMailV31(&messages)
    if err != nil {
        errorLog.Printf("Mailjet API error: %v", err)
        return "", fmt.Errorf("failed to send email: %v", err)
    }
    
    // Log the Mailjet response
    infoLog.Printf("Mailjet response: %+v", res)

    return "Email sent successfully", nil
}

func handleICS(w http.ResponseWriter, data map[string]interface{}, menuData *menu.Menu) error {
    buildingID := data["buildingId"].(string)
    districtID := data["districtId"].(string)
    startDate := data["startDate"].(string)
    endDate := data["endDate"].(string)
    
    infoLog.Printf("Generating ICS file for buildingID: %s, districtID: %s, startDate: %s, endDate: %s", 
                   buildingID, districtID, startDate, endDate)

    icsContent, err := ics.GenerateICSFile(buildingID, districtID, startDate, endDate, "", false)
    if err != nil {
        errorLog.Printf("Failed to generate ICS file: %v", err)
        return fmt.Errorf("failed to generate ICS file: %w", err)
    }

    infoLog.Printf("ICS file generated successfully, content length: %d bytes", len(icsContent))

    filename := fmt.Sprintf("lunch_menu_%s_to_%s.ics", startDate, endDate)

    infoLog.Printf("Setting headers: Content-Type: %s, Content-Disposition: %s, Content-Length: %d",
        "text/calendar", fmt.Sprintf("attachment; filename=\"%s\"", filename), len(icsContent))

    w.Header().Set("Content-Type", "text/calendar")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
    w.Header().Set("Content-Length", fmt.Sprintf("%d", len(icsContent)))
    
    n, err := w.Write(icsContent)
    if err != nil {
        errorLog.Printf("Error writing ICS content to response: %v", err)
        return fmt.Errorf("error writing ICS content to response: %w", err)
    }
    infoLog.Printf("Wrote %d bytes to response", n)

    // Log the first 100 characters of the ICS content
    if len(icsContent) > 100 {
        infoLog.Printf("First 100 characters of ICS content: %s", string(icsContent[:100]))
    } else {
        infoLog.Printf("ICS content: %s", string(icsContent))
    }

    return nil
}
