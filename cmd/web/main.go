package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/asachs01/school_menu_connector/internal/ics"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	// Initialize and configure logrus
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
}

func main() {
	// Create a new ServeMux
	mux := http.NewServeMux()

	// Set up the HTTP server with explicit routes
	mux.HandleFunc("/get-menu", logMiddleware(getMenuHandler))
	mux.HandleFunc("/menu", logMiddleware(serveMenuForm))
	mux.HandleFunc("/", logMiddleware(serveIndex))

	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Infof("Server is running on http://localhost:%s", port)
	logger.Infof("Routes: / and /get-menu are registered")
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	}).Info("Received request to serve index")

	if r.URL.Path != "/" {
		logger.Warn("Not found")
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join("web", "index.html"))
}

// Add this struct for JSON requests
type MenuRequest struct {
	BuildingID   string   `json:"buildingId"`
	DistrictID   string   `json:"districtId"`
	StartDate    string   `json:"startDate"`
	EndDate      string   `json:"endDate"`
	MealTypes    []string `json:"mealTypes"`
}

func getMenuHandler(w http.ResponseWriter, r *http.Request) {
	logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	}).Info("Received request to /get-menu")

	if r.Method != http.MethodPost {
		logger.Warn("Method not allowed")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusMethodNotAllowed)
		message := "Error: Only POST requests are supported for this endpoint."
		w.Write([]byte(message))
		return
	}

	var buildingID, districtID, startDate, endDate string
	var mealTypes []string
	contentType := r.Header.Get("Content-Type")

	// Handle JSON requests
	if contentType == "application/json" {
		var req MenuRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			logger.WithError(err).Error("Error parsing JSON request")
			http.Error(w, "Error parsing JSON request", http.StatusBadRequest)
			return
		}
		buildingID = req.BuildingID
		districtID = req.DistrictID
		startDate = req.StartDate
		endDate = req.EndDate
		mealTypes = req.MealTypes
	} else {
		// Handle form data requests
		if err := r.ParseForm(); err != nil {
			logger.WithError(err).Error("Error parsing form data")
			http.Error(w, "Error parsing form data", http.StatusBadRequest)
			return
		}
		buildingID = r.Form.Get("buildingId")
		districtID = r.Form.Get("districtId")
		startDate = r.Form.Get("startDate")
		endDate = r.Form.Get("endDate")
		mealTypes = r.Form["mealTypes"]
	}

	// If no meal types selected, default to Lunch
	if len(mealTypes) == 0 {
		mealTypes = []string{"Lunch"}
	}

	// Validate required fields
	if buildingID == "" || districtID == "" || startDate == "" || endDate == "" {
		logger.Error("Missing required fields")
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Validate and convert dates
	startDate, err := validateAndConvertDate(startDate)
	if err != nil {
		logger.WithError(err).Error("Invalid start date")
		http.Error(w, fmt.Sprintf("Invalid start date: %v", err), http.StatusBadRequest)
		return
	}

	endDate, err = validateAndConvertDate(endDate)
	if err != nil {
		logger.WithError(err).Error("Invalid end date")
		http.Error(w, fmt.Sprintf("Invalid end date: %v", err), http.StatusBadRequest)
		return
	}

	logger.WithFields(logrus.Fields{
		"buildingID": buildingID,
		"districtID": districtID,
		"startDate":  startDate,
		"endDate":    endDate,
	}).Info("Generating ICS file")

	// Generate the ICS file
	icsContent, err := ics.GenerateICSFileWithMealTypes(buildingID, districtID, startDate, endDate, mealTypes, false)
	if err != nil {
		logger.WithError(err).Error("Error generating ICS file")
		http.Error(w, fmt.Sprintf("Error generating ICS file: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Infof("ICS file generated successfully, content length: %d bytes", len(icsContent))

	// Set response headers
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Content-Disposition", "attachment; filename=school_menu.ics")

	// Write response
	if _, err = w.Write(icsContent); err != nil {
		logger.WithError(err).Error("Error writing response")
	} else {
		logger.Info("ICS file sent successfully")
	}
}

func validateAndConvertDate(date string) (string, error) {
	// Try parsing as yyyy-mm-dd
	if _, err := time.Parse("2006-01-02", date); err == nil {
		// Convert to mm-dd-yyyy
		t, _ := time.Parse("2006-01-02", date)
		return t.Format("01-02-2006"), nil
	}

	// Try parsing as mm-dd-yyyy
	if _, err := time.Parse("01-02-2006", date); err == nil {
		return date, nil
	}

	return "", fmt.Errorf("invalid date format: %s", date)
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

func serveMenuForm(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("web", "menu_form.html"))
}
