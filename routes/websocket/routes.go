package websocket

import (
	"katkam/features/connectivity/relay"
	"net/http"
)

type WebSocketRouter struct {
	relay *relay.WebRTCRelay
}

func NewWebSocketRouter(relay *relay.WebRTCRelay) *WebSocketRouter {
	return &WebSocketRouter{
		relay: relay,
	}
}

func (w *WebSocketRouter) SetupRoutes() {
	http.HandleFunc("/ws/receiver", w.relay.HandleReceiverSignaling)
	http.HandleFunc("/ws/sender", w.relay.HandleSenderSignaling)
}
