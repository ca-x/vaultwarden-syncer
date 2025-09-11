package icons

import (
	"fmt"
)

// Example demonstrates how to use the icon system
func Example() {
	// Create icon manager
	iconManager, err := New()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Get a simple icon
	dashboardIcon := iconManager.Get("dashboard")
	fmt.Printf("Dashboard icon: %s\n", dashboardIcon)

	// Get icon with custom class
	settingsIcon := iconManager.GetWithClass("settings", "icon-large")
	fmt.Printf("Settings icon with class: %s\n", settingsIcon)

	// List all available icons
	fmt.Println("Available icons:")
	for _, name := range iconManager.List() {
		fmt.Printf("- %s\n", name)
	}

	// Check if icon exists
	if iconManager.Exists("dashboard") {
		fmt.Println("Dashboard icon exists!")
	}

	// Usage in templates:
	// {{ icon "dashboard" }}
	// {{ iconWithClass "settings" "icon-large" }}
}

// GetIconHTML returns an icon as HTML string (for testing)
func GetIconHTML(name string) string {
	iconManager, err := New()
	if err != nil {
		return ""
	}
	return string(iconManager.Get(name))
}
