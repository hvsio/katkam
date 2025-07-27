package relay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type WebRTCRelay struct {
	sender   Sender
	receiver Receiver
	mutex    sync.RWMutex
	isActive bool
}

func NewWebRTCRelay(receiver Receiver, sender Sender) *WebRTCRelay {
	relay := &WebRTCRelay{
		receiver: receiver,
		sender:   sender,
	}

	relay.receiver.AssignVideoFrameCallback(relay.relayVideoFrame)
	relay.receiver.AssignAudioFrameCallback(relay.relayAudioFrame)
	relay.receiver.AssignConnectedCallback(relay.onReceiverConnected)
	relay.receiver.AssignDisconnectedCallback(relay.onReceiverDisconnected)

	return relay
}

func (s *WebRTCRelay) Setup() {
	if s.receiver.ExposesReceivingEndpoint() {
		http.HandleFunc("/ws/receiver", s.HandleReceiverSignaling)
	}
	if s.sender.ExposesReceivingEndpoint() {
		http.HandleFunc("/ws/sender", s.HandleSenderSignaling)
	}

	http.HandleFunc("/api/status", s.HandleStatus)
}

func (s *WebRTCRelay) Start() error {
	err := s.receiver.Start()
	if err != nil {
		return err
	}
	err = s.sender.Start()
	return err
}

// HandleReceiverSignaling handles WebSocket connections for the input stream
func (s *WebRTCRelay) HandleReceiverSignaling(w http.ResponseWriter, r *http.Request) {
	s.GetReceiver().HandleWebSocketConnection(w, r)
}

// HandleSenderSignaling handles WebSocket connections for the output stream to frontend
func (s *WebRTCRelay) HandleSenderSignaling(w http.ResponseWriter, r *http.Request) {
	s.GetSender().HandleWebSocketConnection(w, r)
}

// HandleStatus returns the current status of the relay
func (s *WebRTCRelay) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	status := s.GetStatus()
	json.NewEncoder(w).Encode(status)
}

// GetRelay returns the underlying relay instance
func (r *WebRTCRelay) relayVideoFrame(data []byte) {
	if r.sender.IsConnected() {
		r.sender.SendVideoFrame(data)
	}
}

func (r *WebRTCRelay) relayAudioFrame(data []byte) {
	if r.sender.IsConnected() {
		r.sender.SendAudioFrame(data)
	}
}

func (r *WebRTCRelay) onReceiverConnected() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.isActive = true
	fmt.Println("WebRTC Relay: Receiver connected, relay is now active")
}

func (r *WebRTCRelay) onReceiverDisconnected() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.isActive = false
	fmt.Println("WebRTC Relay: Receiver disconnected, relay is now inactive")
}

func (r *WebRTCRelay) GetReceiver() Socket {
	return r.receiver
}

func (r *WebRTCRelay) GetSender() Socket {
	return r.sender
}

func (r *WebRTCRelay) IsActive() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.isActive
}

func (r *WebRTCRelay) GetStatus() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return map[string]interface{}{
		"relay_active":       r.isActive,
		"receiver_connected": r.receiver.IsConnected(),
		"sender_connected":   r.sender.IsConnected(),
	}
}

func (r *WebRTCRelay) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.isActive = false

	var receiverErr, senderErr error
	receiverErr = r.receiver.Close()
	senderErr = r.sender.Close()

	if receiverErr != nil {
		return fmt.Errorf("receiver close error: %v", receiverErr)
	}
	if senderErr != nil {
		return fmt.Errorf("sender close error: %v", senderErr)
	}

	return nil
}
