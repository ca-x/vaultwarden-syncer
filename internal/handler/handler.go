package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ca-x/vaultwarden-syncer/ent"
	"github.com/ca-x/vaultwarden-syncer/ent/storage"
	"github.com/ca-x/vaultwarden-syncer/ent/syncjob"
	"github.com/ca-x/vaultwarden-syncer/internal/cleanup"
	"github.com/ca-x/vaultwarden-syncer/internal/i18n"
	"github.com/ca-x/vaultwarden-syncer/internal/scheduler"
	"github.com/ca-x/vaultwarden-syncer/internal/service"
	"github.com/ca-x/vaultwarden-syncer/internal/setup"
	"github.com/ca-x/vaultwarden-syncer/internal/sync"
	"github.com/ca-x/vaultwarden-syncer/internal/template"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	userService      *service.UserService
	setupService     *setup.SetupService
	syncService      *sync.Service
	cleanupService   *cleanup.Service
	schedulerService *scheduler.Service
	client           *ent.Client
	tmplManager      *template.Manager
}

func New(userService *service.UserService, setupService *setup.SetupService, syncService *sync.Service, cleanupService *cleanup.Service, schedulerService *scheduler.Service, client *ent.Client) *Handler {
	tmplManager, err := template.New()
	if err != nil {
		// Log error but don't fail, fallback to basic responses
		fmt.Printf("Failed to create template manager: %v\n", err)
		tmplManager = nil
	}

	return &Handler{
		userService:      userService,
		setupService:     setupService,
		syncService:      syncService,
		cleanupService:   cleanupService,
		schedulerService: schedulerService,
		client:           client,
		tmplManager:      tmplManager,
	}
}

func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) Index(c echo.Context) error {
	// Authentication middleware already checks setup status,
	// so we can directly show the dashboard
	if h.tmplManager == nil {
		return c.String(http.StatusInternalServerError, "Template manager not available")
	}

	// Get language and translator from context
	lang := i18n.GetLanguageFromContext(c.Request().Context())
	translator := i18n.GetTranslatorFromContext(c.Request().Context())
	if translator == nil {
		translator = i18n.New()
	}

	// Get storage count
	storageCount, err := h.client.Storage.Query().Count(c.Request().Context())
	if err != nil {
		storageCount = 0 // fallback on error
	}

	// Get last sync job with detailed status
	lastSyncTime := translator.T(lang, "time.never")
	syncStatus := translator.T(lang, "status.no_sync")
	syncStatusClass := "icon-info"
	syncStatusIcon := "mdi:information"
	lastSyncError := ""

	lastJob, err := h.client.SyncJob.Query().
		WithStorage().
		Order(ent.Desc(syncjob.FieldCreatedAt)).
		First(c.Request().Context())

	if err == nil {
		lastSyncTime = lastJob.CreatedAt.Format("2006-01-02 15:04")

		switch lastJob.Status {
		case syncjob.StatusCompleted:
			syncStatus = translator.T(lang, "status.sync_success")
			syncStatusClass = "icon-success"
			syncStatusIcon = "mdi:check-circle"
		case syncjob.StatusFailed:
			syncStatus = translator.T(lang, "status.sync_failed")
			syncStatusClass = "icon-danger"
			syncStatusIcon = "mdi:alert-circle"
			if lastJob.Message != "" {
				lastSyncError = lastJob.Message
			}
		case syncjob.StatusRunning:
			syncStatus = translator.T(lang, "status.sync_running")
			syncStatusClass = "icon-warning"
			syncStatusIcon = "mdi:sync"
		case syncjob.StatusPending:
			syncStatus = translator.T(lang, "status.sync_pending")
			syncStatusClass = "icon-info"
			syncStatusIcon = "mdi:clock"
		}
	}

	// Get sync jobs count
	jobsCount, err := h.client.SyncJob.Query().Count(c.Request().Context())
	if err != nil {
		jobsCount = 0
	}

	dashboardData := template.DashboardData{
		StorageCount:    storageCount,
		LastSync:        lastSyncTime,
		BackupSize:      translator.T(lang, "status.calculating"),
		TotalBackups:    jobsCount,
		SystemStatus:    translator.T(lang, "status.operational"),
		SyncStatus:      syncStatus,
		SyncStatusClass: syncStatusClass,
		SyncStatusIcon:  syncStatusIcon,
		LastSyncError:   lastSyncError,
	}

	html, err := h.tmplManager.RenderDashboard(dashboardData, lang, translator)
	if err != nil {
		fmt.Printf("Failed to render dashboard: %v\n", err)
		return c.String(http.StatusInternalServerError, "Failed to render dashboard: "+err.Error())
	}

	return c.HTML(http.StatusOK, html)
}

func (h *Handler) Setup(c echo.Context) error {
	setupComplete, err := h.setupService.IsSetupComplete(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check setup status"})
	}

	if setupComplete {
		return c.Redirect(http.StatusFound, "/")
	}

	if h.tmplManager == nil {
		return c.String(http.StatusInternalServerError, "Template manager not available")
	}

	// Get language and translator from context
	lang := i18n.GetLanguageFromContext(c.Request().Context())
	translator := i18n.GetTranslatorFromContext(c.Request().Context())
	if translator == nil {
		translator = i18n.New()
	}

	html, err := h.tmplManager.RenderSetup(lang, translator)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to render setup page")
	}

	return c.HTML(http.StatusOK, html)
}

func (h *Handler) CompleteSetup(c echo.Context) error {
	var data setup.SetupData
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form data"})
	}

	if err := h.setupService.CompleteSetup(c.Request().Context(), data); err != nil {
		return c.HTML(http.StatusBadRequest, `<div style="color: red;">Error: `+err.Error()+`</div>`)
	}

	return c.HTML(http.StatusOK, `
		<div style="color: green;">
			<p>Setup completed successfully!</p>
			<p><a href="/">Continue to Dashboard</a></p>
		</div>
	`)
}

func (h *Handler) Login(c echo.Context) error {
	// Check if setup is complete first
	setupComplete, err := h.setupService.IsSetupComplete(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check setup status"})
	}

	// If setup is not complete, redirect to setup page
	if !setupComplete {
		return c.Redirect(http.StatusFound, "/setup")
	}

	if h.tmplManager == nil {
		return c.String(http.StatusInternalServerError, "Template manager not available")
	}

	// Get language and translator from context
	lang := i18n.GetLanguageFromContext(c.Request().Context())
	translator := i18n.GetTranslatorFromContext(c.Request().Context())
	if translator == nil {
		translator = i18n.New()
	}

	html, err := h.tmplManager.RenderLogin(lang, translator)
	if err != nil {
		fmt.Printf("Failed to render login page: %v\n", err)
		return c.String(http.StatusInternalServerError, "Failed to render login page: "+err.Error())
	}

	return c.HTML(http.StatusOK, html)
}

func (h *Handler) HandleLogin(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	// Get language and translator from context
	lang := i18n.GetLanguageFromContext(c.Request().Context())
	translator := i18n.GetTranslatorFromContext(c.Request().Context())
	if translator == nil {
		translator = i18n.New()
	}

	if username == "" || password == "" {
		if h.tmplManager == nil {
			return c.HTML(http.StatusBadRequest, `<div class="result error">Username and password are required</div>`)
		}

		message := h.tmplManager.CreateMessage("error", translator.T(lang, "auth.error.missing_fields"))
		html, err := h.tmplManager.RenderLoginWithMessage(lang, translator, message)
		if err != nil {
			return c.HTML(http.StatusBadRequest, `<div class="result error">Username and password are required</div>`)
		}
		return c.HTML(http.StatusBadRequest, html)
	}

	token, user, err := h.userService.Authenticate(c.Request().Context(), username, password)
	if err != nil {
		if h.tmplManager == nil {
			return c.HTML(http.StatusUnauthorized, `<div class="result error">Invalid credentials</div>`)
		}

		message := h.tmplManager.CreateMessage("error", translator.T(lang, "auth.error.invalid_credentials"))
		html, err := h.tmplManager.RenderLoginWithMessage(lang, translator, message)
		if err != nil {
			return c.HTML(http.StatusUnauthorized, `<div class="result error">Invalid credentials</div>`)
		}
		return c.HTML(http.StatusUnauthorized, html)
	}

	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)

	// Use HTMX redirect header to navigate to dashboard
	c.Response().Header().Set("HX-Redirect", "/")

	if h.tmplManager == nil {
		return c.HTML(http.StatusOK, `<div class="result success">Welcome, `+user.Username+`! Redirecting...</div>`)
	}

	message := h.tmplManager.CreateMessage("success", translator.T(lang, "auth.success.welcome", user.Username))
	html, err := h.tmplManager.RenderLoginWithMessage(lang, translator, message)
	if err != nil {
		return c.HTML(http.StatusOK, `<div class="result success">Welcome, `+user.Username+`! Redirecting...</div>`)
	}
	return c.HTML(http.StatusOK, html)
}

// Logout handles user logout
func (h *Handler) Logout(c echo.Context) error {
	// Clear the auth cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Delete the cookie
	}
	c.SetCookie(cookie)
	return c.Redirect(http.StatusFound, "/login")
}

// StorageList displays the storage management page
func (h *Handler) StorageList(c echo.Context) error {
	storages, err := h.client.Storage.Query().All(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load storages"})
	}

	if h.tmplManager == nil {
		return c.String(http.StatusInternalServerError, "Template manager not available")
	}

	// Get language and translator from context
	lang := i18n.GetLanguageFromContext(c.Request().Context())
	translator := i18n.GetTranslatorFromContext(c.Request().Context())
	if translator == nil {
		translator = i18n.New()
	}

	html, err := h.tmplManager.RenderStorage(storages, h.client, lang, translator)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to render storage page")
	}

	return c.HTML(http.StatusOK, html)
}

// Settings displays the settings page
func (h *Handler) Settings(c echo.Context) error {
	if h.tmplManager == nil {
		return c.String(http.StatusInternalServerError, "Template manager not available")
	}

	// Get language and translator from context
	lang := i18n.GetLanguageFromContext(c.Request().Context())
	translator := i18n.GetTranslatorFromContext(c.Request().Context())
	if translator == nil {
		translator = i18n.New()
	}

	html, err := h.tmplManager.RenderSettings(lang, translator)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to render settings page")
	}

	return c.HTML(http.StatusOK, html)
}

// CreateStorage creates a new storage backend
func (h *Handler) CreateStorage(c echo.Context) error {
	// Parse form data
	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form data"})
	}

	name := c.FormValue("name")
	storageType := c.FormValue("type")
	enabled := c.FormValue("enabled") == "on"
	configJSON := c.FormValue("config")

	// Validate required fields
	if name == "" || storageType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Name and type are required"})
	}

	// Parse config JSON
	var config map[string]interface{}
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid config JSON"})
		}
	} else {
		// If no config JSON provided, build config from form fields
		config = make(map[string]interface{})

		if storageType == "webdav" {
			config["url"] = c.FormValue("webdav_url")
			config["username"] = c.FormValue("webdav_username")
			config["password"] = c.FormValue("webdav_password")

			// Validate WebDAV required fields
			if config["url"] == "" || config["username"] == "" || config["password"] == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "WebDAV requires URL, username, and password"})
			}
		} else if storageType == "s3" {
			config["endpoint"] = c.FormValue("s3_endpoint")
			config["access_key_id"] = c.FormValue("s3_access_key_id")
			config["secret_access_key"] = c.FormValue("s3_secret_access_key")
			config["region"] = c.FormValue("s3_region")
			config["bucket"] = c.FormValue("s3_bucket")

			// Validate S3 required fields (endpoint is optional)
			if config["access_key_id"] == "" || config["secret_access_key"] == "" ||
				config["region"] == "" || config["bucket"] == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "S3 requires access key ID, secret access key, region, and bucket"})
			}
		}
	}

	_, err := h.client.Storage.
		Create().
		SetName(name).
		SetType(storage.Type(storageType)).
		SetConfig(config).
		SetEnabled(enabled).
		Save(c.Request().Context())

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create storage: " + err.Error()})
	}

	// Reload storage list to show the new storage
	storages, err := h.client.Storage.Query().All(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load storages"})
	}

	// Get language and translator from context
	lang := i18n.GetLanguageFromContext(c.Request().Context())
	translator := i18n.GetTranslatorFromContext(c.Request().Context())
	if translator == nil {
		translator = i18n.New()
	}

	// Render updated storage cards
	if h.tmplManager != nil {
		storageCards, err := h.tmplManager.RenderStorageCards(storages, h.client, lang, translator)
		if err == nil {
			return c.HTML(http.StatusOK, fmt.Sprintf(`
				<div style="color: green;">Storage created successfully!</div>
				<script>
					// Reset form
					document.getElementById('storage-form').reset();
					// Hide storage type fields
					document.querySelectorAll('.storage-type-fields').forEach(function(el) {
						el.style.display = 'none';
					});
					// Update storage list
					document.getElementById('storage-list').innerHTML = %q;
				</script>
			`, storageCards))
		}
	}

	return c.HTML(http.StatusOK, `<div style="color: green;">Storage created successfully!</div>`)
}

// UpdateStorage updates an existing storage backend
func (h *Handler) UpdateStorage(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid storage ID"})
	}

	// Parse form data
	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid form data"})
	}

	name := c.FormValue("name")
	storageType := c.FormValue("type")
	enabled := c.FormValue("enabled") == "on"
	configJSON := c.FormValue("config")

	// Validate required fields
	if name == "" || storageType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Name and type are required"})
	}

	// Parse config JSON
	var config map[string]interface{}
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid config JSON"})
		}
	} else {
		// If no config JSON provided, build config from form fields
		config = make(map[string]interface{})

		if storageType == "webdav" {
			config["url"] = c.FormValue("webdav_url")
			config["username"] = c.FormValue("webdav_username")
			config["password"] = c.FormValue("webdav_password")

			// Validate WebDAV required fields
			if config["url"] == "" || config["username"] == "" || config["password"] == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "WebDAV requires URL, username, and password"})
			}
		} else if storageType == "s3" {
			config["endpoint"] = c.FormValue("s3_endpoint")
			config["access_key_id"] = c.FormValue("s3_access_key_id")
			config["secret_access_key"] = c.FormValue("s3_secret_access_key")
			config["region"] = c.FormValue("s3_region")
			config["bucket"] = c.FormValue("s3_bucket")

			// Validate S3 required fields (endpoint is optional)
			if config["access_key_id"] == "" || config["secret_access_key"] == "" ||
				config["region"] == "" || config["bucket"] == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "S3 requires access key ID, secret access key, region, and bucket"})
			}
		}
	}

	_, err = h.client.Storage.
		UpdateOneID(id).
		SetName(name).
		SetType(storage.Type(storageType)).
		SetConfig(config).
		SetEnabled(enabled).
		SetUpdatedAt(time.Now()).
		Save(c.Request().Context())

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update storage: " + err.Error()})
	}

	// Reload storage list to show the updated storage
	storages, err := h.client.Storage.Query().All(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load storages"})
	}

	// Get language and translator from context
	lang := i18n.GetLanguageFromContext(c.Request().Context())
	translator := i18n.GetTranslatorFromContext(c.Request().Context())
	if translator == nil {
		translator = i18n.New()
	}

	// Render updated storage cards
	if h.tmplManager != nil {
		storageCards, err := h.tmplManager.RenderStorageCards(storages, h.client, lang, translator)
		if err == nil {
			return c.HTML(http.StatusOK, fmt.Sprintf(`
				<div style="color: green;">Storage updated successfully!</div>
				<script>
					// Update storage list
					document.getElementById('storage-list').innerHTML = %q;
				</script>
			`, storageCards))
		}
	}

	return c.HTML(http.StatusOK, `<div style="color: green;">Storage updated successfully!</div>`)
}

// DeleteStorage deletes a storage backend
func (h *Handler) DeleteStorage(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid storage ID"})
	}

	err = h.client.Storage.DeleteOneID(id).Exec(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete storage"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Storage deleted successfully"})
}

// TriggerSync manually triggers a sync for a specific storage
func (h *Handler) TriggerSync(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.HTML(http.StatusBadRequest, `<div class="result error">Invalid storage ID</div>`)
	}

	// Check if storage exists and is enabled
	storage, err := h.client.Storage.Get(c.Request().Context(), id)
	if err != nil {
		return c.HTML(http.StatusNotFound, `<div class="result error">Storage not found</div>`)
	}

	if !storage.Enabled {
		return c.HTML(http.StatusBadRequest, `<div class="result error">Storage is disabled</div>`)
	}

	// Start sync in background
	go func() {
		ctx := context.Background() // Use background context for async operation
		if err := h.syncService.SyncToStorage(ctx, id); err != nil {
			// Log error for debugging
			fmt.Printf("Sync failed for storage %d: %v\n", id, err)
		}
	}()

	return c.HTML(http.StatusOK, `<div class="result success">
		<iconify-icon icon="mdi:sync" class="icon-success"></iconify-icon>
		Sync triggered successfully! Check the dashboard for progress.
	</div>`)
}

// TriggerConcurrentSync 手动触发并发同步到所有启用的存储后端
func (h *Handler) TriggerConcurrentSync(c echo.Context) error {
	// 获取所有启用的存储后端
	storages, err := h.client.Storage.Query().
		Where(storage.Enabled(true)).
		All(c.Request().Context())

	if err != nil {
		return c.HTML(http.StatusInternalServerError, `<div class="result error">Failed to load storage backends</div>`)
	}

	if len(storages) == 0 {
		return c.HTML(http.StatusBadRequest, `<div class="result error">No enabled storage backends found</div>`)
	}

	// 收集存储ID
	storageIDs := make([]int, len(storages))
	for i, st := range storages {
		storageIDs[i] = st.ID
	}

	// 在后台启动并发同步
	go func() {
		ctx := context.Background()
		if err := h.syncService.ConcurrentSyncToStorages(ctx, storageIDs); err != nil {
			// 记录错误日志
			fmt.Printf("Concurrent sync failed: %v\n", err)
		}
	}()

	return c.HTML(http.StatusOK, `<div class="result success">
		<iconify-icon icon="mdi:sync" class="icon-success"></iconify-icon>
		Concurrent sync triggered successfully! Check the dashboard for progress.
	</div>`)
}

// HealthCheckAll 执行所有存储后端的健康检查
func (h *Handler) HealthCheckAll(c echo.Context) error {
	results := h.schedulerService.HealthCheckAll(c.Request().Context())

	var failed []string
	var passed []string

	for storage, err := range results {
		if err != nil {
			failed = append(failed, fmt.Sprintf("- %s: %v", storage, err))
		} else {
			passed = append(passed, fmt.Sprintf("- %s: OK", storage))
		}
	}

	// 准备响应消息
	var message strings.Builder
	if len(failed) > 0 {
		message.WriteString("Failed storage backends:\n")
		message.WriteString(strings.Join(failed, "\n"))
		message.WriteString("\n\n")
	}

	if len(passed) > 0 {
		message.WriteString("Healthy storage backends:\n")
		message.WriteString(strings.Join(passed, "\n"))
		message.WriteString("\n")
	}

	// 如果有失败的存储后端，返回错误状态
	if len(failed) > 0 {
		return c.HTML(http.StatusOK, fmt.Sprintf(`<div class="result error">
			<iconify-icon icon="mdi:alert-circle" class="icon-danger"></iconify-icon>
			Health check completed with %d failed backend(s)<br><br>
			<pre>%s</pre>
		</div>`, len(failed), message.String()))
	}

	return c.HTML(http.StatusOK, fmt.Sprintf(`<div class="result success">
		<iconify-icon icon="mdi:check-circle" class="icon-success"></iconify-icon>
		All storage backends are healthy (%d passed)<br><br>
		<pre>%s</pre>
	</div>`, len(passed), message.String()))
}

// GetSyncJobs returns the list of sync jobs
func (h *Handler) GetSyncJobs(c echo.Context) error {
	jobs, err := h.client.SyncJob.
		Query().
		WithStorage().
		Order(ent.Desc(syncjob.FieldCreatedAt)).
		Limit(50).
		All(c.Request().Context())

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load sync jobs"})
	}

	return c.JSON(http.StatusOK, jobs)
}

// GetSyncStatus returns the current sync status for dashboard
func (h *Handler) GetSyncStatus(c echo.Context) error {
	// Get language and translator from context
	lang := i18n.GetLanguageFromContext(c.Request().Context())
	translator := i18n.GetTranslatorFromContext(c.Request().Context())
	if translator == nil {
		translator = i18n.New()
	}

	// Get last sync job with detailed status
	syncStatus := translator.T(lang, "status.no_sync")
	syncStatusClass := "icon-info"
	syncStatusIcon := "mdi:information"
	lastSyncTime := translator.T(lang, "time.never")
	lastSyncError := ""

	lastJob, err := h.client.SyncJob.Query().
		WithStorage().
		Order(ent.Desc(syncjob.FieldCreatedAt)).
		First(c.Request().Context())

	if err == nil {
		lastSyncTime = lastJob.CreatedAt.Format("2006-01-02 15:04")

		switch lastJob.Status {
		case syncjob.StatusCompleted:
			syncStatus = translator.T(lang, "status.sync_success")
			syncStatusClass = "icon-success"
			syncStatusIcon = "mdi:check-circle"
		case syncjob.StatusFailed:
			syncStatus = translator.T(lang, "status.sync_failed")
			syncStatusClass = "icon-danger"
			syncStatusIcon = "mdi:alert-circle"
			if lastJob.Message != "" {
				lastSyncError = lastJob.Message
			}
		case syncjob.StatusRunning:
			syncStatus = translator.T(lang, "status.sync_running")
			syncStatusClass = "icon-warning"
			syncStatusIcon = "mdi:sync"
		case syncjob.StatusPending:
			syncStatus = translator.T(lang, "status.sync_pending")
			syncStatusClass = "icon-info"
			syncStatusIcon = "mdi:clock"
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":      syncStatus,
		"statusClass": syncStatusClass,
		"statusIcon":  syncStatusIcon,
		"lastSync":    lastSyncTime,
		"error":       lastSyncError,
	})
}

// TriggerCleanup manually triggers cleanup of old sync job records
func (h *Handler) TriggerCleanup(c echo.Context) error {
	go func() {
		ctx := context.Background()
		if err := h.schedulerService.RunCleanupNow(ctx); err != nil {
			fmt.Printf("Manual cleanup failed: %v\n", err)
		}
	}()

	return c.HTML(http.StatusOK, `<div class="result success">
		<iconify-icon icon="mdi:broom" class="icon-success"></iconify-icon>
		Cleanup triggered successfully! Old sync records are being removed.
	</div>`)
}

// GetSyncJobStats returns statistics about sync job records
func (h *Handler) GetSyncJobStats(c echo.Context) error {
	stats, err := h.cleanupService.GetSyncJobStats(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get sync job statistics"})
	}

	return c.JSON(http.StatusOK, stats)
}
