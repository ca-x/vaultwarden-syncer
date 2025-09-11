package icons

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
)

//go:embed *.svg
var IconFS embed.FS

// IconManager manages SVG icons
type IconManager struct {
	icons map[string]string
}

// New creates a new icon manager
func New() (*IconManager, error) {
	manager := &IconManager{
		icons: make(map[string]string),
	}
	
	// Load all icons from embedded filesystem
	entries, err := IconFS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read icon directory: %w", err)
	}
	
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".svg") {
			content, err := IconFS.ReadFile(entry.Name())
			if err != nil {
				return nil, fmt.Errorf("failed to read icon %s: %w", entry.Name(), err)
			}
			
			// Remove .svg extension for icon name
			iconName := strings.TrimSuffix(entry.Name(), ".svg")
			manager.icons[iconName] = string(content)
		}
	}
	
	return manager, nil
}

// Get returns an icon as HTML template
func (im *IconManager) Get(name string) template.HTML {
	if svg, exists := im.icons[name]; exists {
		return template.HTML(svg)
	}
	// Return a default icon or empty if not found
	return template.HTML(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>`)
}

// GetWithClass returns an icon with custom CSS classes
func (im *IconManager) GetWithClass(name, class string) template.HTML {
	if svg, exists := im.icons[name]; exists {
		// Add class to the SVG element
		if class != "" {
			svg = strings.Replace(svg, "<svg", fmt.Sprintf(`<svg class="%s"`, class), 1)
		}
		return template.HTML(svg)
	}
	return im.Get(name) // fallback to default
}

// List returns all available icon names
func (im *IconManager) List() []string {
	var names []string
	for name := range im.icons {
		names = append(names, name)
	}
	return names
}

// Exists checks if an icon exists
func (im *IconManager) Exists(name string) bool {
	_, exists := im.icons[name]
	return exists
}