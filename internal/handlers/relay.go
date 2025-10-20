package handlers

import (
	"fmt"
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
	rh.relay.Start()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func (rh *RelayHandler) Stop(w http.ResponseWriter, r *http.Request) {
	rh.relay.Close()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func (rh *RelayHandler) HandleReceiverSignaling(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling receiver signaling")
	rh.relay.GetReceiver().StartWebSocketConnection(w, r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func (rh *RelayHandler) HandleSenderSignaling(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling sender signaling")
	rh.relay.GetSender().StartWebSocketConnection(w, r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}
