package main

import (
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
	mux.HandleFunc("/", logMiddleware(serveIndex))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Infof("Server is running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join("web", "index.html"))
}

func getMenuHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("Received request to /get-menu")

	if r.Method != http.MethodPost {
		logger.Warn("Method not allowed")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusMethodNotAllowed)
		message := "Error: Only POST requests are supported for this endpoint. Please use a POST request with the required form data to generate an ICS file."
		w.Write([]byte(message))
		return
	}

	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		logger.WithError(err).Error("Error parsing form data")
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	// Get parameters from the form data
	buildingID := r.Form.Get("buildingId")
	districtID := r.Form.Get("districtId")
	startDate := r.Form.Get("startDate")
	endDate := r.Form.Get("endDate")

	// Validate and convert dates if necessary
	startDate, err = validateAndConvertDate(startDate)
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

	// Generate the ICS file based on the parameters
	icsContent, err := ics.GenerateICSFile(buildingID, districtID, startDate, endDate, "", false)
	if err != nil {
		logger.WithError(err).Error("Error generating ICS file")
		http.Error(w, fmt.Sprintf("Error generating ICS file: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Infof("ICS file generated successfully, content length: %d bytes", len(icsContent))

	// Set the content type and headers for file download
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Content-Disposition", "attachment; filename=school_menu.ics")

	// Write the ICS data to the response
	_, err = w.Write(icsContent)
	if err != nil {
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