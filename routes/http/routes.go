package http

import (
	"katkam/handlers"
	"net/http"
)

type HttpRouter struct {
	authController *handlers.AuthController
}

func NewHttpRouter(authController *handlers.AuthController) *HttpRouter {
	return &HttpRouter{
		authController: authController,
	}
}

func (h *HttpRouter) SetupRoutes() {
	http.HandleFunc("/auth/login", h.authController.Login)
	http.HandleFunc("/auth/logout", h.authController.Logout)
}
