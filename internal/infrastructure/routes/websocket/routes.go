package websocket

import (
	"katkam/internal/handlers"
	"net/http"
)

type WebSocketRouter struct {
	relayHandler *handlers.RelayHandler
}

func NewWebSocketRouter(relayHandler *handlers.RelayHandler) *WebSocketRouter {
	return &WebSocketRouter{
		relayHandler: relayHandler,
	}
}

func (w *WebSocketRouter) SetupRoutes() {
	http.HandleFunc("/ws/receiver", w.relayHandler.HandleReceiverSignaling)
	http.HandleFunc("/ws/sender", w.relayHandler.HandleSenderSignaling)
}
