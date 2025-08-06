package handlers

import (
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

func (rh *RelayHandler) Start(w http.ResponseWriter, r *http.Request) {
	rh.relay.Start()
}

func (rh *RelayHandler) HandleReceiverSignaling(w http.ResponseWriter, r *http.Request) {
	rh.relay.GetReceiver().HandleWebSocketConnection(w, r)
}

func (rh *RelayHandler) HandleSenderSignaling(w http.ResponseWriter, r *http.Request) {
	rh.relay.GetSender().HandleWebSocketConnection(w, r)
}
