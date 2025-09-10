package server

import (
	"fmt"
	"vaultwarden-syncer/internal/config"
	"vaultwarden-syncer/internal/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo   *echo.Echo
	config *config.Config
}

func New(cfg *config.Config, handler *handler.Handler) *Server {
	e := echo.New()
	
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/", handler.Index)
	e.GET("/health", handler.Health)
	e.GET("/setup", handler.Setup)
	e.POST("/api/setup", handler.CompleteSetup)
	e.GET("/login", handler.Login)
	e.POST("/api/login", handler.HandleLogin)

	return &Server{
		echo:   e,
		config: cfg,
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	return s.echo.Start(addr)
}

func (s *Server) Shutdown() error {
	return s.echo.Shutdown(nil)
}