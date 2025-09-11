package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ca-x/vaultwarden-syncer/internal/auth"
	"github.com/ca-x/vaultwarden-syncer/internal/config"
	"github.com/ca-x/vaultwarden-syncer/internal/handler"
	"github.com/ca-x/vaultwarden-syncer/internal/i18n"
	"github.com/ca-x/vaultwarden-syncer/internal/middleware"
	"github.com/ca-x/vaultwarden-syncer/internal/setup"
	"github.com/ca-x/vaultwarden-syncer/internal/template"

	"github.com/labstack/echo/v4"
	echo_middleware "github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo   *echo.Echo
	config *config.Config
}

func New(cfg *config.Config, handler *handler.Handler, authService *auth.Service, setupService *setup.SetupService) *Server {
	e := echo.New()

	e.Use(echo_middleware.Logger())
	e.Use(echo_middleware.Recover())
	e.Use(echo_middleware.CORS())

	// I18n middleware
	translator := i18n.New()
	e.Use(i18n.Middleware(translator))

	// Static file server for CSS and other assets
	e.GET("/static/*", func(c echo.Context) error {
		path := c.Param("*")
		tmplManager, err := template.New()
		if err != nil {
			return c.String(http.StatusInternalServerError, "Template manager not available")
		}

		reader, err := tmplManager.ServeStatic(path)
		if err != nil {
			return c.String(http.StatusNotFound, "File not found")
		}

		// Set appropriate content type based on file extension
		contentType := getContentType(path)
		c.Response().Header().Set("Content-Type", contentType)
		return c.Stream(http.StatusOK, contentType, reader)
	})

	// Authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(authService, setupService)

	// Public routes (no authentication required)
	e.GET("/health", handler.Health)
	e.GET("/setup", handler.Setup)
	e.POST("/api/setup", handler.CompleteSetup)
	e.GET("/login", handler.Login)
	e.POST("/api/login", handler.HandleLogin)

	// Protected routes (authentication required)
	protected := e.Group("", authMiddleware.RequireAuth())
	protected.GET("/", handler.Index)
	protected.GET("/logout", handler.Logout)
	protected.GET("/storage", handler.StorageList)
	protected.GET("/settings", handler.Settings)
	protected.POST("/api/storage", handler.CreateStorage)
	protected.PUT("/api/storage/:id", handler.UpdateStorage)
	protected.DELETE("/api/storage/:id", handler.DeleteStorage)
	protected.POST("/api/sync/:id", handler.TriggerSync)
	protected.POST("/api/sync-concurrent", handler.TriggerConcurrentSync) // 添加并发同步端点
	protected.POST("/api/health-check", handler.HealthCheckAll)           // 添加健康检查端点
	protected.GET("/api/jobs", handler.GetSyncJobs)
	protected.GET("/api/sync/status", handler.GetSyncStatus)
	protected.POST("/api/cleanup", handler.TriggerCleanup)
	protected.GET("/api/stats", handler.GetSyncJobStats)

	return &Server{
		echo:   e,
		config: cfg,
	}
}

// getContentType returns the appropriate content type based on file extension
func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".html", ".htm":
		return "text/html"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "text/plain"
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	return s.echo.Start(addr)
}

func (s *Server) Shutdown() error {
	return s.echo.Shutdown(nil)
}
