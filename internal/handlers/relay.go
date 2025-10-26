package handlers

import (
	"katkam/internal/infrastructure/websockets/relay"
	"net/http"
)

type RelayHandler struct {
	relay *relay.WebRTCRelay
}

func NewRelayHandler(relay *relay.WebRTCRelay) *RelayHandler {
	return &RelayHandler{
		relay: relay,
	}
}

// Starts the receiver and sender components
// Also begins to listen for video and audio frames from the receiver channels and sends them to the sender
func (rh *RelayHandler) Start(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(http.StatusOK)
	if r.Method == "GET" {
		rh.relay.Start(r.Context())
	}
}

func (rh *RelayHandler) Stop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "GET" {
		rh.relay.Close()
	}
	w.WriteHeader(http.StatusOK)
}

func (rh *RelayHandler) HandleReceiverSignaling(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	rh.relay.GetReceiver().HandleWebSocketConnection(w, r)
}

func (rh *RelayHandler) HandleSenderSignaling(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	rh.relay.GetSender().HandleWebSocketConnection(w, r)
}
