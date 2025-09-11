package template

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"sync"

	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/internal/i18n"
)

//go:embed web/*
var Assets embed.FS

// Manager handles template rendering
type Manager struct {
	templates *template.Template
}

// Singleton instance for template manager
var templateManager *Manager
var initOnce sync.Once

// NavItem represents a navigation item
type NavItem struct {
	URL  string
	Icon string
	Text string
}

// PageData represents the data passed to templates
type PageData struct {
	Title      string
	AuthLayout bool
	ShowNav    bool
	NavItems   []NavItem
	Content    template.HTML
	Data       interface{}
	Lang       i18n.Language
	T          func(string, ...interface{}) string
}

// StorageCardData represents storage card data
type StorageCardData struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	StatusColor string                 `json:"status_color"`
	Created     string                 `json:"created"`
	Icon        string                 `json:"icon"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// DashboardData represents dashboard statistics
type DashboardData struct {
	StorageCount int
	LastSync     string
	BackupSize   string
	TotalBackups int
	SystemStatus string
}

// New creates a new template manager with singleton pattern for efficiency
func New() (*Manager, error) {
	var err error
	initOnce.Do(func() {
		tmpl, e := template.ParseFS(Assets, "web/*.html")
		if e != nil {
			err = fmt.Errorf("failed to parse templates: %w", e)
			return
		}
		templateManager = &Manager{
			templates: tmpl,
		}
	})

	if err != nil {
		return nil, err
	}

	return templateManager, nil
}

// RenderDashboard renders the dashboard page
func (m *Manager) RenderDashboard(data DashboardData, lang i18n.Language, translator *i18n.Translator) (string, error) {
	// Create template data with translations
	templateData := struct {
		DashboardData
		T func(string, ...interface{}) string
	}{
		DashboardData: data,
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	var content bytes.Buffer
	err := m.templates.ExecuteTemplate(&content, "dashboard.html", templateData)
	if err != nil {
		return "", err
	}

	pageData := PageData{
		Title:      translator.T(lang, "dashboard.title"),
		AuthLayout: false,
		ShowNav:    true,
		NavItems: []NavItem{
			{URL: "/", Icon: "🏠", Text: translator.T(lang, "nav.dashboard")},
			{URL: "/storage", Icon: "💾", Text: translator.T(lang, "nav.storage")},
			{URL: "/settings", Icon: "⚙️", Text: translator.T(lang, "nav.settings")},
			{URL: "/logout", Icon: "🚪", Text: translator.T(lang, "nav.logout")},
		},
		Content: template.HTML(content.String()),
		Lang:    lang,
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	return m.renderLayout(pageData)
}

// RenderLogin renders the login page
func (m *Manager) RenderLogin(lang i18n.Language, translator *i18n.Translator) (string, error) {
	// Create template data with translations
	templateData := struct {
		T func(string, ...interface{}) string
	}{
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	var content bytes.Buffer
	err := m.templates.ExecuteTemplate(&content, "login.html", templateData)
	if err != nil {
		return "", err
	}

	pageData := PageData{
		Title:      translator.T(lang, "auth.login.title"),
		AuthLayout: true,
		ShowNav:    false,
		Content:    template.HTML(content.String()),
		Lang:       lang,
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	return m.renderLayout(pageData)
}

// RenderSetup renders the setup page
func (m *Manager) RenderSetup(lang i18n.Language, translator *i18n.Translator) (string, error) {
	// Create template data with translations
	templateData := struct {
		T func(string, ...interface{}) string
	}{
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	var content bytes.Buffer
	err := m.templates.ExecuteTemplate(&content, "setup.html", templateData)
	if err != nil {
		return "", err
	}

	pageData := PageData{
		Title:      translator.T(lang, "setup.title"),
		AuthLayout: true,
		ShowNav:    false,
		Content:    template.HTML(content.String()),
		Lang:       lang,
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	return m.renderLayout(pageData)
}

// RenderStorage renders the storage management page
func (m *Manager) RenderStorage(storages []*ent.Storage, lang i18n.Language, translator *i18n.Translator) (string, error) {
	// Render storage cards
	storageCards, err := m.RenderStorageCards(storages, lang, translator)
	if err != nil {
		return "", err
	}

	var content bytes.Buffer
	data := struct {
		StorageCards template.HTML
		T            func(string, ...interface{}) string
	}{
		StorageCards: template.HTML(storageCards),
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	err = m.templates.ExecuteTemplate(&content, "storage.html", data)
	if err != nil {
		return "", err
	}

	pageData := PageData{
		Title:      translator.T(lang, "storage.title"),
		AuthLayout: false,
		ShowNav:    true,
		NavItems: []NavItem{
			{URL: "/", Icon: "🏠", Text: translator.T(lang, "nav.dashboard")},
			{URL: "/settings", Icon: "⚙️", Text: translator.T(lang, "nav.settings")},
			{URL: "/logout", Icon: "🚪", Text: translator.T(lang, "nav.logout")},
		},
		Content: template.HTML(content.String()),
		Lang:    lang,
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	return m.renderLayout(pageData)
}

// RenderSettings renders the settings page
func (m *Manager) RenderSettings(lang i18n.Language, translator *i18n.Translator) (string, error) {
	// Create template data with translations
	templateData := struct {
		T func(string, ...interface{}) string
	}{
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	var content bytes.Buffer
	err := m.templates.ExecuteTemplate(&content, "settings.html", templateData)
	if err != nil {
		return "", err
	}

	pageData := PageData{
		Title:      translator.T(lang, "settings.title"),
		AuthLayout: false,
		ShowNav:    true,
		NavItems: []NavItem{
			{URL: "/", Icon: "🏠", Text: translator.T(lang, "nav.dashboard")},
			{URL: "/storage", Icon: "💾", Text: translator.T(lang, "nav.storage")},
			{URL: "/logout", Icon: "🚪", Text: translator.T(lang, "nav.logout")},
		},
		Content: template.HTML(content.String()),
		Lang:    lang,
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	return m.renderLayout(pageData)
}

// RenderStorageCards renders storage cards
func (m *Manager) RenderStorageCards(storages []*ent.Storage, lang i18n.Language, translator *i18n.Translator) (string, error) {
	var cards []StorageCardData

	for _, s := range storages {
		status := translator.T(lang, "storage.disabled")
		statusColor := "var(--apple-red)"
		if s.Enabled {
			status = translator.T(lang, "storage.enabled")
			statusColor = "var(--apple-green)"
		}

		typeIcon := "🌐"
		if string(s.Type) == "s3" {
			typeIcon = "☁️"
		}

		cards = append(cards, StorageCardData{
			ID:          s.ID,
			Name:        s.Name,
			Type:        string(s.Type),
			Status:      status,
			StatusColor: statusColor,
			Created:     s.CreatedAt.Format("2006-01-02 15:04"),
			Icon:        typeIcon,
			Config:      s.Config,
		})
	}

	// Create template data with translations
	templateData := struct {
		Cards []StorageCardData
		T     func(string, ...interface{}) string
	}{
		Cards: cards,
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	var content bytes.Buffer
	err := m.templates.ExecuteTemplate(&content, "storage-cards.html", templateData)
	if err != nil {
		return "", err
	}

	return content.String(), nil
}

// renderLayout renders the main layout with content
func (m *Manager) renderLayout(data PageData) (string, error) {
	var output bytes.Buffer
	err := m.templates.ExecuteTemplate(&output, "layout.html", data)
	if err != nil {
		return "", err
	}
	return output.String(), nil
}

// ServeStatic serves static files from embedded templates
func (m *Manager) ServeStatic(path string) (io.Reader, error) {
	// Support various static file types from templates directory
	filePath := "web/" + path

	// Check if the file exists in the embedded filesystem
	file, err := Assets.Open(filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}
