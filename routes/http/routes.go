package http

import (
	"katkam/controllers"
	"net/http"
)

type HttpRouter struct {
	authController *controllers.AuthController
}

func NewHttpRouter(authController *controllers.AuthController) *HttpRouter {
	return &HttpRouter{
		authController: authController,
	}
}

func (h *HttpRouter) SetupRoutes() {
	http.HandleFunc("/auth/login", h.authController.Login)
	http.HandleFunc("/auth/logout", h.authController.Logout)
}
