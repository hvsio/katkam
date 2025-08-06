package relay

import (
	"fmt"
	"sync"

	"katkam/internal/infrastructure/connectivity"
)

type WebRTCRelay struct {
	sender   connectivity.Sender
	receiver connectivity.Receiver
	mutex    sync.RWMutex
	isActive bool
}

func NewWebRTCRelay(receiver connectivity.Receiver, sender connectivity.Sender) *WebRTCRelay {
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

func (s *WebRTCRelay) Start() {
	err := s.receiver.Start()
	if err != nil {
		panic(err)
	}
	err = s.sender.Start()
	if err != nil {
		panic(err)
	}
}

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

func (r *WebRTCRelay) GetReceiver() connectivity.Socket {
	return r.receiver
}

func (r *WebRTCRelay) GetSender() connectivity.Socket {
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
