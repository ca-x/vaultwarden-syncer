package handler

import (
	"net/http"
	"vaultwarden-syncer/internal/service"
	"vaultwarden-syncer/internal/setup"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	userService  *service.UserService
	setupService *setup.SetupService
}

func New(userService *service.UserService, setupService *setup.SetupService) *Handler {
	return &Handler{
		userService:  userService,
		setupService: setupService,
	}
}

func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *Handler) Index(c echo.Context) error {
	setupComplete, err := h.setupService.IsSetupComplete(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check setup status"})
	}

	if !setupComplete {
		return c.Redirect(http.StatusFound, "/setup")
	}

	return c.HTML(http.StatusOK, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Vaultwarden Syncer</title>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<link rel="stylesheet" href="https://unpkg.com/@picocss/pico@latest/css/pico.min.css">
			<script src="https://unpkg.com/htmx.org@1.9.10"></script>
		</head>
		<body>
			<main class="container">
				<nav>
					<ul>
						<li><strong>Vaultwarden Syncer</strong></li>
					</ul>
					<ul>
						<li><a href="/logout">Logout</a></li>
					</ul>
				</nav>
				<h1>Dashboard</h1>
				<div class="grid">
					<section>
						<h2>Sync Status</h2>
						<p>All systems operational</p>
					</section>
					<section>
						<h2>Storage</h2>
						<p>Configure your storage backends</p>
						<a href="/storage" role="button">Manage Storage</a>
					</section>
				</div>
			</main>
		</body>
		</html>
	`)
}

func (h *Handler) Setup(c echo.Context) error {
	setupComplete, err := h.setupService.IsSetupComplete(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check setup status"})
	}

	if setupComplete {
		return c.Redirect(http.StatusFound, "/")
	}

	return c.HTML(http.StatusOK, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Setup - Vaultwarden Syncer</title>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<link rel="stylesheet" href="https://unpkg.com/@picocss/pico@latest/css/pico.min.css">
			<script src="https://unpkg.com/htmx.org@1.9.10"></script>
		</head>
		<body>
			<main class="container">
				<h1>Initial Setup</h1>
				<p>Welcome! Let's set up your Vaultwarden Syncer instance.</p>
				
				<form hx-post="/api/setup" hx-target="#result">
					<div class="grid">
						<label for="admin_username">Admin Username</label>
						<input type="text" id="admin_username" name="admin_username" required>
					</div>
					
					<div class="grid">
						<label for="admin_password">Admin Password</label>
						<input type="password" id="admin_password" name="admin_password" required minlength="8">
					</div>
					
					<div class="grid">
						<label for="admin_email">Admin Email (optional)</label>
						<input type="email" id="admin_email" name="admin_email">
					</div>
					
					<button type="submit">Complete Setup</button>
				</form>
				
				<div id="result"></div>
			</main>
		</body>
		</html>
	`)
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
	return c.HTML(http.StatusOK, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Login - Vaultwarden Syncer</title>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<link rel="stylesheet" href="https://unpkg.com/@picocss/pico@latest/css/pico.min.css">
			<script src="https://unpkg.com/htmx.org@1.9.10"></script>
		</head>
		<body>
			<main class="container">
				<h1>Login</h1>
				
				<form hx-post="/api/login" hx-target="#result">
					<div class="grid">
						<label for="username">Username</label>
						<input type="text" id="username" name="username" required>
					</div>
					
					<div class="grid">
						<label for="password">Password</label>
						<input type="password" id="password" name="password" required>
					</div>
					
					<button type="submit">Login</button>
				</form>
				
				<div id="result"></div>
			</main>
		</body>
		</html>
	`)
}

func (h *Handler) HandleLogin(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		return c.HTML(http.StatusBadRequest, `<div style="color: red;">Username and password are required</div>`)
	}

	token, user, err := h.userService.Authenticate(c.Request().Context(), username, password)
	if err != nil {
		return c.HTML(http.StatusUnauthorized, `<div style="color: red;">Invalid credentials</div>`)
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

	return c.HTML(http.StatusOK, `
		<div style="color: green;">
			<p>Welcome, `+user.Username+`!</p>
			<p><a href="/">Go to Dashboard</a></p>
		</div>
	`)
}