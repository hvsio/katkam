package http

import (
	"katkam/internal/handlers"
	"net/http"
)

type HttpRouter struct {
	authHandler  *handlers.AuthHandler
	relayHandler *handlers.RelayHandler
}

func NewHttpRouter(authHandler *handlers.AuthHandler, relayHandler *handlers.RelayHandler) *HttpRouter {
	return &HttpRouter{
		authHandler:  authHandler,
		relayHandler: relayHandler,
	}
}

func (h *HttpRouter) SetupRoutes() {
	http.HandleFunc("/relay/start", h.relayHandler.Start)

	http.HandleFunc("/auth/login", h.authHandler.Login)
	http.HandleFunc("/auth/logout", h.authHandler.Logout)
	http.HandleFunc("/auth/validate", h.authHandler.ValidateToken)
}
