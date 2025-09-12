package template

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"sync"

	"github.com/ca-x/vaultwarden-syncer/ent"
	storage "github.com/ca-x/vaultwarden-syncer/ent/storage"
	"github.com/ca-x/vaultwarden-syncer/ent/syncjob"
	"github.com/ca-x/vaultwarden-syncer/internal/i18n"
	"github.com/ca-x/vaultwarden-syncer/internal/icons"
)

//go:embed web/*
var Assets embed.FS

// Manager handles template rendering
type Manager struct {
	templates   *template.Template
	iconManager *icons.IconManager
}

// Singleton instance for template manager
var templateManager *Manager
var initOnce sync.Once

// NavItem represents a navigation item
type NavItem struct {
	URL  string
	Icon template.HTML
	Text string
}

// Message represents a user message
type Message struct {
	Type    string // success, error, warning, info
	Content string
	Icon    template.HTML
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
	Message    *Message
}

// StorageCardData represents storage card data
type StorageCardData struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Status          string                 `json:"status"`
	StatusColor     string                 `json:"status_color"`
	Created         string                 `json:"created"`
	Icon            template.HTML          `json:"icon"`
	Config          map[string]interface{} `json:"config,omitempty"`
	LastSync        string                 `json:"last_sync"`
	LastSyncStatus  string                 `json:"last_sync_status"`
	SyncStatusIcon  string                 `json:"sync_status_icon"`
	SyncStatusClass string                 `json:"sync_status_class"`
	SyncError       string                 `json:"sync_error,omitempty"`
}

// DashboardData represents dashboard statistics
type DashboardData struct {
	StorageCount    int
	LastSync        string
	BackupSize      string
	TotalBackups    int
	SystemStatus    string
	SyncStatus      string
	SyncStatusClass string
	SyncStatusIcon  string
	LastSyncError   string
}

// New creates a new template manager with singleton pattern for efficiency
func New() (*Manager, error) {
	var err error
	initOnce.Do(func() {
		iconMgr, e := icons.New()
		if e != nil {
			err = fmt.Errorf("failed to initialize icon manager: %w", e)
			return
		}

		// Create template functions
		funcMap := template.FuncMap{
			"icon":          iconMgr.Get,
			"iconWithClass": iconMgr.GetWithClass,
		}

		tmpl, e := template.New("").Funcs(funcMap).ParseFS(Assets, "web/*.html")
		if e != nil {
			err = fmt.Errorf("failed to parse templates: %w", e)
			return
		}

		templateManager = &Manager{
			templates:   tmpl,
			iconManager: iconMgr,
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
			{URL: "/", Icon: m.Icon("dashboard"), Text: translator.T(lang, "nav.dashboard")},
			{URL: "/storage", Icon: m.Icon("database"), Text: translator.T(lang, "nav.storage")},
			{URL: "/settings", Icon: m.Icon("settings"), Text: translator.T(lang, "nav.settings")},
			{URL: "/system-info", Icon: m.Icon("information"), Text: translator.T(lang, "nav.system_info")},
			{URL: "/logout", Icon: m.Icon("logout"), Text: translator.T(lang, "nav.logout")},
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
	return m.RenderLoginWithMessage(lang, translator, nil)
}

// RenderLoginWithMessage renders the login page with a message
func (m *Manager) RenderLoginWithMessage(lang i18n.Language, translator *i18n.Translator, message *Message) (string, error) {
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
		Message:    message,
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
func (m *Manager) RenderStorage(storages []*ent.Storage, client *ent.Client, lang i18n.Language, translator *i18n.Translator) (string, error) {
	// Render storage cards
	storageCards, err := m.RenderStorageCards(storages, client, lang, translator)
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
        return "", fmt.Errorf("template execution failed: %w", err)
    }

	pageData := PageData{
		Title:      translator.T(lang, "storage.title"),
		AuthLayout: false,
		ShowNav:    true,
		NavItems: []NavItem{
			{URL: "/", Icon: m.Icon("dashboard"), Text: translator.T(lang, "nav.dashboard")},
			{URL: "/storage", Icon: m.Icon("database"), Text: translator.T(lang, "nav.storage")},
			{URL: "/settings", Icon: m.Icon("settings"), Text: translator.T(lang, "nav.settings")},
			{URL: "/system-info", Icon: m.Icon("information"), Text: translator.T(lang, "nav.system_info")},
			{URL: "/logout", Icon: m.Icon("logout"), Text: translator.T(lang, "nav.logout")},
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
			{URL: "/", Icon: m.Icon("dashboard"), Text: translator.T(lang, "nav.dashboard")},
			{URL: "/storage", Icon: m.Icon("database"), Text: translator.T(lang, "nav.storage")},
			{URL: "/settings", Icon: m.Icon("settings"), Text: translator.T(lang, "nav.settings")},
			{URL: "/system-info", Icon: m.Icon("information"), Text: translator.T(lang, "nav.system_info")},
			{URL: "/logout", Icon: m.Icon("logout"), Text: translator.T(lang, "nav.logout")},
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
func (m *Manager) RenderStorageCards(storages []*ent.Storage, client *ent.Client, lang i18n.Language, translator *i18n.Translator) (string, error) {
	var cards []StorageCardData

	for _, s := range storages {
		status := translator.T(lang, "storage.disabled")
		statusColor := "var(--apple-red)"
		if s.Enabled {
			status = translator.T(lang, "storage.enabled")
			statusColor = "var(--apple-green)"
		}

		typeIcon := m.Icon("web")
		if string(s.Type) == "s3" {
			typeIcon = m.Icon("aws")
		}

		// Get last sync info for this storage
		lastSync := translator.T(lang, "time.never")
		lastSyncStatus := translator.T(lang, "status.no_sync")
		syncStatusIcon := "info"
		syncStatusClass := "icon-info"
		syncError := ""

		if client != nil {
			ctx := context.Background()
			lastJob, err := client.SyncJob.Query().
				Where(syncjob.HasStorageWith(storage.IDEQ(s.ID))).
				Order(ent.Desc(syncjob.FieldCreatedAt)).
				First(ctx)

			if err == nil {
				lastSync = lastJob.CreatedAt.Format("2006-01-02 15:04")

				switch lastJob.Status {
				case syncjob.StatusCompleted:
					lastSyncStatus = translator.T(lang, "status.sync_success")
					syncStatusClass = "icon-success"
					syncStatusIcon = "check-circle"
				case syncjob.StatusFailed:
					lastSyncStatus = translator.T(lang, "status.sync_failed")
					syncStatusClass = "icon-danger"
					syncStatusIcon = "alert-circle"
					if lastJob.Message != "" {
						syncError = lastJob.Message
					}
				case syncjob.StatusRunning:
					lastSyncStatus = translator.T(lang, "status.sync_running")
					syncStatusClass = "icon-warning"
					syncStatusIcon = "sync"
				case syncjob.StatusPending:
					lastSyncStatus = translator.T(lang, "status.sync_pending")
					syncStatusClass = "icon-info"
					syncStatusIcon = "clock"
				}
			}
		}

		// Prepare config data based on storage type
		config := make(map[string]interface{})

		// Load the storage with its config edges
		if client != nil {
			ctx := context.Background()
			loadedStorage, err := client.Storage.Query().
				Where(storage.IDEQ(s.ID)).
				WithWebdavConfig().
				WithS3Config().
				Only(ctx)

			if err == nil {
				if loadedStorage.Edges.WebdavConfig != nil {
					config["url"] = loadedStorage.Edges.WebdavConfig.URL
					config["username"] = loadedStorage.Edges.WebdavConfig.Username
				} else if loadedStorage.Edges.S3Config != nil {
					config["endpoint"] = loadedStorage.Edges.S3Config.Endpoint
					config["access_key_id"] = loadedStorage.Edges.S3Config.AccessKeyID
					config["secret_access_key"] = loadedStorage.Edges.S3Config.SecretAccessKey
					config["region"] = loadedStorage.Edges.S3Config.Region
					config["bucket"] = loadedStorage.Edges.S3Config.Bucket
				}
			}
		}

		cards = append(cards, StorageCardData{
			ID:              s.ID,
			Name:            s.Name,
			Type:            string(s.Type),
			Status:          status,
			StatusColor:     statusColor,
			Created:         s.CreatedAt.Format("2006-01-02 15:04"),
			Icon:            typeIcon,
			Config:          config,
			LastSync:        lastSync,
			LastSyncStatus:  lastSyncStatus,
			SyncStatusIcon:  syncStatusIcon,
			SyncStatusClass: syncStatusClass,
			SyncError:       syncError,
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
        return "", fmt.Errorf("storage-cards template execution failed: %w", err)
    }

    result := content.String()
    return result, nil
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

// RenderLayout renders the main layout with content
func (m *Manager) RenderLayout(data PageData) (string, error) {
	return m.renderLayout(data)
}

// ExecuteTemplate executes a template with the given name
func (m *Manager) ExecuteTemplate(buf *bytes.Buffer, name string, data interface{}) error {
	return m.templates.ExecuteTemplate(buf, name, data)
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

// Icon returns an SVG icon by name
func (m *Manager) Icon(name string) template.HTML {
	return m.iconManager.Get(name)
}

// IconWithClass returns an SVG icon with custom CSS class
func (m *Manager) IconWithClass(name, class string) template.HTML {
	return m.iconManager.GetWithClass(name, class)
}

// CreateMessage creates a message with appropriate icon
func (m *Manager) CreateMessage(msgType, content string) *Message {
	var icon template.HTML
	switch msgType {
	case "success":
		icon = m.Icon("check-circle")
	case "error":
		icon = m.Icon("alert-circle")
	case "warning":
		icon = m.Icon("alert-circle")
	case "info":
		icon = m.Icon("info")
	default:
		icon = m.Icon("info")
	}

	return &Message{
		Type:    msgType,
		Content: content,
		Icon:    icon,
	}
}

// RenderSystemInfo renders the system information page
func (m *Manager) RenderSystemInfo(systemInfo map[string]interface{}, lang i18n.Language, translator *i18n.Translator) (string, error) {
	// Create template data with translations
	templateData := struct {
		Version            string
		BuildDate          string
		GitCommit          string
		GoVersion          string
		Platform           string
		Uptime             string
		DatabaseType       string
		DatabaseSize       string
		DatabasePath       string
		VaultwardenDataPath string
		VaultwardenDataSize string
		LastBackupTime     string
		T                  func(string, ...interface{}) string
	}{
		Version:            systemInfo["version"].(string),
		BuildDate:          systemInfo["build_date"].(string),
		GitCommit:          systemInfo["git_commit"].(string),
		GoVersion:          systemInfo["go_version"].(string),
		Platform:           systemInfo["platform"].(string),
		Uptime:             systemInfo["uptime"].(string),
		DatabaseType:       "SQLite", // Default to SQLite as per project requirements
		DatabaseSize:       "N/A",
		DatabasePath:       "N/A",
		VaultwardenDataPath: "N/A",
		VaultwardenDataSize: "N/A", 
		LastBackupTime:     "N/A",
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	var content bytes.Buffer
	err := m.templates.ExecuteTemplate(&content, "system-info.html", templateData)
	if err != nil {
		return "", err
	}

	pageData := PageData{
		Title:      translator.T(lang, "system.title"),
		AuthLayout: false,
		ShowNav:    true,
		NavItems: []NavItem{
			{URL: "/", Icon: m.Icon("dashboard"), Text: translator.T(lang, "nav.dashboard")},
			{URL: "/storage", Icon: m.Icon("database"), Text: translator.T(lang, "nav.storage")},
			{URL: "/settings", Icon: m.Icon("settings"), Text: translator.T(lang, "nav.settings")},
			{URL: "/system-info", Icon: m.Icon("information"), Text: translator.T(lang, "nav.system_info")},
			{URL: "/logout", Icon: m.Icon("logout"), Text: translator.T(lang, "nav.logout")},
		},
		Content: template.HTML(content.String()),
		Lang:    lang,
		T: func(key string, args ...interface{}) string {
			return translator.T(lang, key, args...)
		},
	}

	return m.renderLayout(pageData)
}
