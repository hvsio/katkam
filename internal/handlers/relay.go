package handlers

import (
	"fmt"
	"katkam/internal/infrastructure/connectivity/relay"
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
}

func (rh *RelayHandler) Stop(w http.ResponseWriter, r *http.Request) {
	rh.relay.Close()
}

func (rh *RelayHandler) HandleReceiverSignaling(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling receiver signaling")
	rh.relay.GetReceiver().StartWebSocketConnection(w, r)
}

func (rh *RelayHandler) HandleSenderSignaling(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling sender signaling")
	rh.relay.GetSender().StartWebSocketConnection(w, r)
}
