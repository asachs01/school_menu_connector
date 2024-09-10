package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/asachs01/school_menu_connector/internal/ics"
	"github.com/asachs01/school_menu_connector/internal/menu"
	mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

// Create a custom logger
var (
    infoLog  = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
    errorLog = log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
)

// Add this new logger
var logger *logrus.Logger

// Create a custom visitor struct which holds the rate limiter for each
// visitor and the last time that the visitor was seen.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Change the map to hold values of the type visitor.
var visitors = make(map[string]*visitor)
var mu sync.Mutex

var sanitizer = bluemonday.UGCPolicy()

func sanitizeInput(input string) string {
	return sanitizer.Sanitize(input)
}

var senderEmail string

func init() {
	senderEmail = os.Getenv("SENDER_EMAIL")
	if senderEmail == "" {
		senderEmail = "noreply@schoolmenuconnector.com"
	}

	// Start the cleanup goroutine
	go cleanupVisitors()

	// Initialize and configure logrus
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
}

func main() {
	// Serve static files
	fs := http.FileServer(http.Dir("./web"))

	// Create a new CORS handler
	c := cors.New(cors.Options{
		AllowedOrigins: getAllowedOrigins(),
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		Debug:          os.Getenv("CORS_DEBUG") == "true",
	})

	// Create a new router
	mux := http.NewServeMux()

	// Add your routes to the new router
	mux.HandleFunc("/", securityHeadersMiddleware(logMiddleware(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				injectRecaptchaKey(w, r)
			} else {
				fs.ServeHTTP(w, r)
			}
		},
	)))

	mux.HandleFunc("/api/generate", securityHeadersMiddleware(logMiddleware(rateLimitMiddleware(
		handleGenerate,
	))))

	// Wrap the router with the CORS handler
	handler := c.Handler(mux)

	// Start the server with the CORS-enabled handler
	infoLog.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", handler)
	errorLog.Fatal(err)
}

func logMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        logger.WithFields(logrus.Fields{
            "method": r.Method,
            "path":   r.URL.Path,
            "ip":     r.RemoteAddr,
        }).Info("Request received")
        next.ServeHTTP(w, r)
    }
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if r := recover(); r != nil {
            logger.WithFields(logrus.Fields{
                "panic": r,
                "stack": string(debug.Stack()),
            }).Error("Panic in handleGenerate")
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

    buildingID := sanitizeInput(data["buildingId"].(string))
    districtID := sanitizeInput(data["districtId"].(string))
    startDate := sanitizeInput(data["startDate"].(string))
    endDate := sanitizeInput(data["endDate"].(string))
    action := sanitizeInput(data["action"].(string))

    // Verify reCAPTCHA
    token, ok := data["recaptchaToken"].(string)
    if !ok {
        errorLog.Print("Missing or invalid reCAPTCHA token")
        http.Error(w, "Missing or invalid reCAPTCHA token", http.StatusBadRequest)
        return
    }
    valid, err := verifyRecaptcha(token)
    if err != nil {
        errorLog.Printf("Error verifying reCAPTCHA: %v", err)
        http.Error(w, "Error verifying reCAPTCHA", http.StatusInternalServerError)
        return
    }
    if !valid {
        errorLog.Print("Invalid reCAPTCHA")
        http.Error(w, "Invalid reCAPTCHA", http.StatusBadRequest)
        return
    }

    menuData, err := menu.Fetch(buildingID, districtID, startDate, endDate, false)
    if err != nil {
        logger.WithFields(logrus.Fields{
            "buildingID": buildingID,
            "districtID": districtID,
            "startDate":  startDate,
            "endDate":    endDate,
            "error":      err,
        }).Error("Error fetching menu")
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
				Name:  "School Menu Connector",
			},
			To:       &recipientsV31,
			Subject:  "School Menu",
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

func injectRecaptchaKey(w http.ResponseWriter, r *http.Request) {
    siteKey := os.Getenv("RECAPTCHA_SITE_KEY")
    if siteKey == "" {
        log.Println("RECAPTCHA_SITE_KEY not set")
        http.Error(w, "RECAPTCHA_SITE_KEY not set", http.StatusInternalServerError)
        return
    }

    html, err := os.ReadFile("web/index.html")
    if err != nil {
        log.Printf("Error reading HTML file: %v", err)
        http.Error(w, "Error reading HTML file", http.StatusInternalServerError)
        return
    }

    modifiedHTML := strings.Replace(string(html), "RECAPTCHA_SITE_KEY", siteKey, -1)

    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(modifiedHTML))

    // Log the first 200 characters of modifiedHTML to check if replacement occurred
    log.Printf("First 200 chars of modified HTML: %s", modifiedHTML[:200])
}

func verifyRecaptcha(token string) (bool, error) {
	secretKey := os.Getenv("RECAPTCHA_SECRET_KEY")
	if secretKey == "" {
		return false, fmt.Errorf("RECAPTCHA_SECRET_KEY not set")
	}
	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify",
		url.Values{"secret": {secretKey}, "response": {token}})
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result["success"].(bool), nil
}

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Every(1*time.Second), 5)
		// Create a new visitor and add it to the map.
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	return v.limiter
}

func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limiter := getVisitor(r.RemoteAddr)
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func getAllowedOrigins() []string {
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins != "" {
		return strings.Split(origins, ",")
	}
	// Default to localhost if no origins are specified
	return []string{"http://localhost:8080"}
}

func securityHeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Security-Policy", 
            "default-src 'self'; " +
            "script-src 'self' https://www.google.com/recaptcha/ https://www.gstatic.com/recaptcha/ https://cdn.jsdelivr.net 'unsafe-inline'; " +
            "style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline'; " +
            "font-src 'self' https://cdn.jsdelivr.net data:; " +
            "frame-src https://www.google.com/recaptcha/; " +
            "connect-src 'self' https://www.google.com/recaptcha/; " +
            "img-src 'self' data:")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        next.ServeHTTP(w, r)
    }
}
