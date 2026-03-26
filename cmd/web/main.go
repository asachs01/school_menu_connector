package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/asachs01/school_menu_connector/internal/cache"
	"github.com/asachs01/school_menu_connector/internal/ics"
	"github.com/asachs01/school_menu_connector/internal/menu"
	"github.com/sirupsen/logrus"
)

var (
	logger    *logrus.Logger
	menuCache *cache.Cache
)

func init() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	var err error
	menuCache, err = cache.New("", 0) // defaults: /tmp/menu-cache, 6h TTL
	if err != nil {
		logger.WithError(err).Warn("Failed to initialize menu cache, continuing without cache")
	}
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/get-menu", logMiddleware(getMenuHandler))
	mux.HandleFunc("/get-menu-json", logMiddleware(getMenuJSONHandler))
	mux.HandleFunc("/menu", logMiddleware(serveMenuForm))
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/refresh-cache", logMiddleware(refreshCacheHandler))
	mux.HandleFunc("/", logMiddleware(serveIndex))

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

// MenuRequest holds the JSON body for menu endpoints.
type MenuRequest struct {
	BuildingID string   `json:"buildingId"`
	DistrictID string   `json:"districtId"`
	StartDate  string   `json:"startDate"`
	EndDate    string   `json:"endDate"`
	MealTypes  []string `json:"mealTypes"`
}

// MenuEvent represents a single calendar event for JSON response.
type MenuEvent struct {
	Date        string `json:"date"`
	MealType    string `json:"mealType"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// MenuEventsResponse is the JSON response for /get-menu-json.
type MenuEventsResponse struct {
	Events []MenuEvent `json:"events"`
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

	if len(mealTypes) == 0 {
		mealTypes = []string{"Lunch"}
	}

	if buildingID == "" || districtID == "" || startDate == "" || endDate == "" {
		logger.Error("Missing required fields")
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

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

	icsContent, err := ics.GenerateICSFileWithMealTypes(buildingID, districtID, startDate, endDate, mealTypes, false)
	if err != nil {
		logger.WithError(err).Error("Error generating ICS file")
		http.Error(w, fmt.Sprintf("Error generating ICS file: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Infof("ICS file generated successfully, content length: %d bytes", len(icsContent))

	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Content-Disposition", "attachment; filename=school_menu.ics")

	if _, err = w.Write(icsContent); err != nil {
		logger.WithError(err).Error("Error writing response")
	} else {
		logger.Info("ICS file sent successfully")
	}
}

func getMenuJSONHandler(w http.ResponseWriter, r *http.Request) {
	logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	}).Info("Received request to /get-menu-json")

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are supported", http.StatusMethodNotAllowed)
		return
	}

	var buildingID, districtID, startDate, endDate string
	var mealTypes []string
	contentType := r.Header.Get("Content-Type")

	if contentType == "application/json" {
		var req MenuRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Error parsing JSON request", http.StatusBadRequest)
			return
		}
		buildingID = req.BuildingID
		districtID = req.DistrictID
		startDate = req.StartDate
		endDate = req.EndDate
		mealTypes = req.MealTypes
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Error parsing form data", http.StatusBadRequest)
			return
		}
		buildingID = r.Form.Get("buildingId")
		districtID = r.Form.Get("districtId")
		startDate = r.Form.Get("startDate")
		endDate = r.Form.Get("endDate")
		mealTypes = r.Form["mealTypes"]
	}

	if len(mealTypes) == 0 {
		mealTypes = []string{"Lunch"}
	}

	if buildingID == "" || districtID == "" || startDate == "" || endDate == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	startDateInternal, err := validateAndConvertDate(startDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid start date: %v", err), http.StatusBadRequest)
		return
	}
	endDateInternal, err := validateAndConvertDate(endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid end date: %v", err), http.StatusBadRequest)
		return
	}

	start, _ := time.Parse("01-02-2006", startDateInternal)
	end, _ := time.Parse("01-02-2006", endDateInternal)

	var events []MenuEvent

	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		dateStr := date.Format("01-02-2006")
		menuData := fetchWithCache(buildingID, districtID, dateStr, dateStr)
		if menuData == nil {
			continue
		}

		for _, mealType := range mealTypes {
			mealMenu := menuData.GetMenuForSession(mealType, date.Format("1/2/2006"), false)
			if mealMenu != "" {
				events = append(events, MenuEvent{
					Date:        date.Format("2006-01-02"),
					MealType:    mealType,
					Title:       fmt.Sprintf("%s Menu - %s", mealType, date.Format("01/02/2006")),
					Description: mealMenu,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MenuEventsResponse{Events: events})
}

// fetchWithCache tries the cache first, then the API, and falls back to
// a stale cache entry if the API fails.
func fetchWithCache(buildingID, districtID, startDate, endDate string) *menu.Menu {
	// Try cache first.
	if menuCache != nil {
		if cached, ok := menuCache.Get(buildingID, districtID, startDate, endDate); ok {
			logger.WithFields(logrus.Fields{
				"buildingID": buildingID,
				"startDate":  startDate,
			}).Debug("Cache hit")
			return cached
		}
	}

	// Fetch from API (may go through proxy if LINQ_PROXY_URL is set).
	menuData, err := menu.Fetch(buildingID, districtID, startDate, endDate, false)
	if err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"buildingID": buildingID,
			"startDate":  startDate,
		}).Warn("API fetch failed")
		return nil
	}

	// Store in cache.
	if menuCache != nil {
		menuCache.Set(buildingID, districtID, startDate, endDate, menuData)
	}

	return menuData
}

// RefreshRequest defines the JSON body for /refresh-cache.
type RefreshRequest struct {
	Schools []struct {
		BuildingID string `json:"buildingId"`
		DistrictID string `json:"districtId"`
	} `json:"schools"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// refreshCacheHandler pre-fetches menus for a list of schools and warms the cache.
// POST /refresh-cache
// Protected by REFRESH_API_KEY env var (if set).
func refreshCacheHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are supported", http.StatusMethodNotAllowed)
		return
	}

	// Check API key if configured.
	if apiKey := os.Getenv("REFRESH_API_KEY"); apiKey != "" {
		provided := r.Header.Get("X-API-Key")
		if provided != apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if len(req.Schools) == 0 || req.StartDate == "" || req.EndDate == "" {
		http.Error(w, "Missing required fields: schools, startDate, endDate", http.StatusBadRequest)
		return
	}

	startDate, err := validateAndConvertDate(req.StartDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid start date: %v", err), http.StatusBadRequest)
		return
	}
	endDate, err := validateAndConvertDate(req.EndDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid end date: %v", err), http.StatusBadRequest)
		return
	}

	cached := 0
	failed := 0

	for _, school := range req.Schools {
		start, _ := time.Parse("01-02-2006", startDate)
		end, _ := time.Parse("01-02-2006", endDate)

		for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
			dateStr := date.Format("01-02-2006")
			menuData, fetchErr := menu.Fetch(school.BuildingID, school.DistrictID, dateStr, dateStr, false)
			if fetchErr != nil {
				logger.WithError(fetchErr).WithFields(logrus.Fields{
					"buildingID": school.BuildingID,
					"date":       dateStr,
				}).Warn("Failed to fetch menu for cache refresh")
				failed++
				continue
			}
			if menuCache != nil {
				menuCache.Set(school.BuildingID, school.DistrictID, dateStr, dateStr, menuData)
			}
			cached++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cached": cached,
		"failed": failed,
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func validateAndConvertDate(date string) (string, error) {
	if _, err := time.Parse("2006-01-02", date); err == nil {
		t, _ := time.Parse("2006-01-02", date)
		return t.Format("01-02-2006"), nil
	}

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
