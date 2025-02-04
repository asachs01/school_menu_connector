package menu

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const apiURL = "https://api.linqconnect.com/api/FamilyMenu"

// Menu represents the structure of the API response
type Menu struct {
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
	AcademicCalendars []interface{} `json:"AcademicCalendars"`
}

// Fetch retrieves the menu from the API for the given building, district, and date range
func Fetch(buildingID, districtID, startDate, endDate string, debug bool) (*Menu, error) {
	url := constructURL(buildingID, districtID, startDate, endDate)
	if debug {
		fmt.Printf("API URL: %s\n", url)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP GET request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if debug {
		fmt.Printf("Response body: %s\n", string(body))
	}

	var menu Menu
	if err := json.Unmarshal(body, &menu); err != nil {
		return nil, fmt.Errorf("unmarshaling JSON: %w", err)
	}

	if debug {
		fmt.Printf("Parsed menu: %+v\n", menu)
	}

	return &menu, nil
}

// constructURL builds the API URL with the given parameters
func constructURL(buildingID, districtID, startDate, endDate string) string {
	return fmt.Sprintf("%s?buildingId=%s&districtId=%s&startDate=%s&endDate=%s", apiURL, buildingID, districtID, startDate, endDate)
}

func (m *Menu) GetLunchMenuString() string {
	var lunchMenu strings.Builder

	for _, session := range m.FamilyMenuSessions {
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

func (m *Menu) GetLunchMenuForDate(date string, debug bool) string {
	var lunchMenu strings.Builder

	if debug {
		fmt.Printf("Searching for lunch menu on date: %s\n", date)
	}

	for _, session := range m.FamilyMenuSessions {
		if session.ServingSession == "Lunch" {
			for _, plan := range session.MenuPlans {
				for _, day := range plan.Days {
					if debug {
						fmt.Printf("Checking day: %s\n", day.Date)
					}

					if day.Date == date {
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

// GetMenuForSession returns the menu string for a specific serving session (Breakfast, Lunch, or Snack)
func (m *Menu) GetMenuForSession(session string, date string, debug bool) string {
	var menuBuilder strings.Builder

	for _, sess := range m.FamilyMenuSessions {
		if sess.ServingSession == session {
			for _, plan := range sess.MenuPlans {
				for _, day := range plan.Days {
					if debug {
						fmt.Printf("Checking day: %s\n", day.Date)
					}

					if day.Date == date {
						fmt.Fprintf(&menuBuilder, "%s Menu for %s:\n\n", session, day.Date)
						for _, meal := range day.MenuMeals {
							fmt.Fprintf(&menuBuilder, "%s:\n", meal.MenuMealName)
							for _, category := range meal.RecipeCategories {
								fmt.Fprintf(&menuBuilder, "  %s:\n", category.CategoryName)
								for _, recipe := range category.Recipes {
									fmt.Fprintf(&menuBuilder, "    - %s\n", recipe.RecipeName)
								}
							}
							menuBuilder.WriteString("\n")
						}
						return menuBuilder.String()
					}
				}
			}
		}
	}

	if debug {
		fmt.Printf("No %s menu found for date: %s\n", session, date)
	}
	return ""
}
