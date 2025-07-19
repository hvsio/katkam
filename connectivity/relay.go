package connectivity

import (
	"fmt"
	"sync"
)

type WebRTCRelay struct {
	receiver *WebRTCReceiver
	sender   *WebRTCSender
	mutex    sync.RWMutex
	isActive bool
}

func NewWebRTCRelay() *WebRTCRelay {
	receiver := NewWebRTCReceiver()
	sender := NewWebRTCSender()
	
	relay := &WebRTCRelay{
		receiver: receiver,
		sender:   sender,
	}
	
	// Set up callbacks to bridge receiver to sender
	receiver.OnVideoFrame = relay.relayVideoFrame
	receiver.OnAudioFrame = relay.relayAudioFrame
	receiver.OnConnected = relay.onReceiverConnected
	receiver.OnDisconnected = relay.onReceiverDisconnected
	
	return relay
}

func (r *WebRTCRelay) relayVideoFrame(data []byte) {
	if r.isActive && r.sender.IsConnected() {
		r.sender.SendVideoFrame(data)
	}
}

func (r *WebRTCRelay) relayAudioFrame(data []byte) {
	if r.isActive && r.sender.IsConnected() {
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

func (r *WebRTCRelay) GetReceiver() *WebRTCReceiver {
	return r.receiver
}

func (r *WebRTCRelay) GetSender() *WebRTCSender {
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
	if r.receiver != nil {
		receiverErr = r.receiver.Close()
	}
	if r.sender != nil {
		senderErr = r.sender.Close()
	}
	
	if receiverErr != nil {
		return fmt.Errorf("receiver close error: %v", receiverErr)
	}
	if senderErr != nil {
		return fmt.Errorf("sender close error: %v", senderErr)
	}
	
	return nil
}
